package marketplace

import (
    "context"
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestRepositoryRequiresHTTPSAndStableIDs(t *testing.T) {
    svc, err := NewService(NewMemoryRepository(), Config{AllowUntrustedRepositories: true, InstallRoot: t.TempDir()})
    require.NoError(t, err)
    _, err = svc.AddRepository(context.Background(), "http://example.com/plugins")
    require.Error(t, err)
    a, err := svc.AddRepository(context.Background(), "https://plugins.example/repository")
    require.NoError(t, err)
    b, err := svc.AddRepository(context.Background(), "https://plugins.example/repository/")
    require.NoError(t, err)
    require.Equal(t, a.ID, b.ID)
}

func TestSyncVerifiesEd25519IndexAndInstallVerifiesArtifactHash(t *testing.T) {
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    require.NoError(t, err)
    artifact := []byte("domus-wasm-module-fixture")
    artifactHash := sha256.Sum256(artifact)
    artifactURL := "https://plugins.example/artifacts/door.wasm"

    index := RepositoryIndex{
        SchemaVersion: 1,
        Repository: IndexRepository{ID: "acme-home", Name: "Acme Home", Owner: "acme", KeyID: "acme-key"},
        Packages: []IndexPackage{{
            ID: "acme.door", Name: "Door bridge", Type: "wasm_plugin", Version: "1.2.0",
            Author: "Acme", Description: "Door bridge", License: "MIT", Runtime: "wasi",
            Capabilities: []string{"events:read"},
            Artifact: Artifact{URL: artifactURL, SHA256: hex.EncodeToString(artifactHash[:]), Size: int64(len(artifact))},
        }},
    }
    indexBody, err := json.Marshal(index)
    require.NoError(t, err)
    signature := ed25519.Sign(privateKey, indexBody)

    mux := http.NewServeMux()
    mux.HandleFunc("/repository/index.json", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(indexBody) })
    mux.HandleFunc("/repository/index.sig", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(signature))) })
    mux.HandleFunc("/artifacts/door.wasm", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(artifact) })
    server := httptest.NewTLSServer(mux)
    defer server.Close()
    trusted := map[string]string{"acme-key": base64.StdEncoding.EncodeToString(publicKey)}
    keysJSON, _ := json.Marshal(trusted)

    repo := NewMemoryRepository()
    client := server.Client()
    svc, err := NewService(repo, Config{
        TrustedKeysJSON: string(keysJSON), InstallRoot: t.TempDir(),
        DomusVersion: "1.0.0", MaxArtifactBytes: 1024, HTTPClient: client,
    })
    require.NoError(t, err)
    source, err := svc.AddRepository(context.Background(), server.URL+"/repository")
    require.NoError(t, err)
    synced, err := svc.SyncRepository(context.Background(), source.ID)
    require.NoError(t, err)
    require.True(t, synced.Trusted)
    pkg, err := repo.GetPackage(context.Background(), "acme.door")
    require.NoError(t, err)
    require.True(t, pkg.Compatible)

    installed, err := svc.InstallArtifact(context.Background(), pkg.ID)
    require.NoError(t, err)
    require.FileExists(t, installed.InstalledPath)
    contents, err := os.ReadFile(installed.InstalledPath)
    require.NoError(t, err)
    require.Equal(t, artifact, contents)

    removed, err := svc.UninstallArtifact(context.Background(), pkg.ID)
    require.NoError(t, err)
    require.Empty(t, removed.InstalledPath)
}

func TestUnsignedRepositoryCanOnlyBeUsedWhenExplicitlyAllowed(t *testing.T) {
    index := RepositoryIndex{
        SchemaVersion: 1,
        Repository: IndexRepository{ID: "unsigned", Name: "Unsigned repository"},
        Packages: []IndexPackage{},
    }
    body, _ := json.Marshal(index)
    mux := http.NewServeMux()
    mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) })
    server := httptest.NewTLSServer(mux)
    defer server.Close()

    sourceRepo := NewMemoryRepository()
    strict, err := NewService(sourceRepo, Config{HTTPClient: server.Client(), InstallRoot: t.TempDir()})
    require.NoError(t, err)
    source, err := strict.AddRepository(context.Background(), server.URL)
    require.NoError(t, err)
    _, err = strict.SyncRepository(context.Background(), source.ID)
    require.ErrorIs(t, err, ErrUntrusted)

    permissiveRepo := NewMemoryRepository()
    permissive, err := NewService(permissiveRepo, Config{
        AllowUntrustedRepositories: true, HTTPClient: server.Client(), InstallRoot: t.TempDir(),
    })
    require.NoError(t, err)
    source, err = permissive.AddRepository(context.Background(), server.URL)
    require.NoError(t, err)
    synced, err := permissive.SyncRepository(context.Background(), source.ID)
    require.NoError(t, err)
    require.False(t, synced.Trusted)
}

func TestArtifactPathCannotEscapeInstallationRoot(t *testing.T) {
    repo := NewMemoryRepository()
    svc, err := NewService(repo, Config{InstallRoot: t.TempDir(), AllowUntrustedRepositories: true})
    require.NoError(t, err)
    outside := filepath.Join(t.TempDir(), "outside.bin")
    require.NoError(t, os.WriteFile(outside, []byte("x"), 0o600))
    now := time.Now()
    require.NoError(t, repo.SaveSource(context.Background(), &Source{ID:"source", URL:"https://plugins.example", Enabled:true}))
    require.NoError(t, repo.SavePackage(context.Background(), &Package{
        ID:"safe.plugin", RepositoryID:"source", Version:"1.0.0",
        Artifact: Artifact{URL:"https://plugins.example/a", SHA256:"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size:1},
        InstalledPath: outside, UpdatedAt: now,
    }))
    _, err = svc.UninstallArtifact(context.Background(), "safe.plugin")
    require.Error(t, err)
    require.FileExists(t, outside)
}
