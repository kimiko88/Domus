// Package marketplace implements a HACS-style, signed repository catalog for Domus plugins.
package marketplace

import (
    "bytes"
    "context"
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "time"
)

var (
    ErrNotFound = errors.New("marketplace: item not found")
    ErrUntrusted = errors.New("marketplace: repository is not trusted")
    ErrSignature = errors.New("marketplace: repository signature verification failed")
    ErrArtifactIntegrity = errors.New("marketplace: artifact SHA-256 mismatch")
    packageIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,127}$`)
    semverPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$`)
    sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

const (
    IndexSchemaVersion = 1
    maxIndexBytes = 1 << 20
    maxPackagesPerIndex = 250
)

type Source struct {
    ID string `json:"id"`
    URL string `json:"url"`
    Name string `json:"name"`
    Owner string `json:"owner,omitempty"`
    Description string `json:"description,omitempty"`
    KeyID string `json:"key_id,omitempty"`
    Trusted bool `json:"trusted"`
    Enabled bool `json:"enabled"`
    IndexSHA256 string `json:"index_sha256,omitempty"`
    LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
    LastError string `json:"last_error,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Artifact struct {
    URL string `json:"url"`
    SHA256 string `json:"sha256"`
    Size int64 `json:"size"`
}

type Package struct {
    ID string `json:"id"`
    RepositoryID string `json:"repository_id"`
    Name string `json:"name"`
    Type string `json:"type"`
    Version string `json:"version"`
    Author string `json:"author"`
    Description string `json:"description"`
    License string `json:"license"`
    Runtime string `json:"runtime"`
    MinDomusVersion string `json:"min_domus_version,omitempty"`
    MaxDomusVersion string `json:"max_domus_version,omitempty"`
    Capabilities []string `json:"capabilities,omitempty"`
    Artifact Artifact `json:"artifact"`
    Trusted bool `json:"trusted"`
    Compatible bool `json:"compatible"`
    InstalledPath string `json:"installed_path,omitempty"`
    InstalledAt *time.Time `json:"installed_at,omitempty"`
    UpdatedAt time.Time `json:"updated_at"`
}

type RepositoryIndex struct {
    SchemaVersion int `json:"schema_version"`
    Repository IndexRepository `json:"repository"`
    Packages []IndexPackage `json:"packages"`
}

type IndexRepository struct {
    ID string `json:"id"`
    Name string `json:"name"`
    Owner string `json:"owner"`
    Description string `json:"description"`
    KeyID string `json:"key_id"`
}

type IndexPackage struct {
    ID string `json:"id"`
    Name string `json:"name"`
    Type string `json:"type"`
    Version string `json:"version"`
    Author string `json:"author"`
    Description string `json:"description"`
    License string `json:"license"`
    Runtime string `json:"runtime"`
    MinDomusVersion string `json:"min_domus_version"`
    MaxDomusVersion string `json:"max_domus_version"`
    Capabilities []string `json:"capabilities"`
    Artifact Artifact `json:"artifact"`
}

type Repository interface {
    SaveSource(context.Context, *Source) error
    GetSource(context.Context, string) (*Source, error)
    ListSources(context.Context) ([]*Source, error)
    DeleteSource(context.Context, string) error
    SavePackage(context.Context, *Package) error
    GetPackage(context.Context, string) (*Package, error)
    ListPackages(context.Context, string) ([]*Package, error)
    DeletePackage(context.Context, string) error
}

type Config struct {
    TrustedKeysJSON string
    AllowUntrustedRepositories bool
    InstallRoot string
    MaxArtifactBytes int64
    DomusVersion string
    HTTPClient *http.Client
}

type Service struct {
    repo Repository
    cfg Config
    keys map[string]ed25519.PublicKey
    httpClient *http.Client
}

func NewService(repo Repository, cfg Config) (*Service, error) {
    if repo == nil {
        return nil, errors.New("marketplace: repository is required")
    }
    if cfg.InstallRoot == "" { cfg.InstallRoot = "./data/marketplace" }
    if cfg.MaxArtifactBytes <= 0 { cfg.MaxArtifactBytes = 16 << 20 }
    if cfg.MaxArtifactBytes > 128<<20 { return nil, errors.New("marketplace: MaxArtifactBytes cannot exceed 128 MiB") }
    if cfg.DomusVersion == "" { cfg.DomusVersion = "1.0.0" }
    if !semverPattern.MatchString(cfg.DomusVersion) { return nil, errors.New("marketplace: invalid Domus version") }
    keys := make(map[string]ed25519.PublicKey)
    if strings.TrimSpace(cfg.TrustedKeysJSON) != "" && strings.TrimSpace(cfg.TrustedKeysJSON) != "{}" {
        var encoded map[string]string
        if err := json.Unmarshal([]byte(cfg.TrustedKeysJSON), &encoded); err != nil {
            return nil, fmt.Errorf("marketplace: parse trusted keys JSON: %w", err)
        }
        if len(encoded) > 64 { return nil, errors.New("marketplace: at most 64 trusted signing keys are supported") }
        for id, value := range encoded {
            if !packageIDPattern.MatchString(strings.ToLower(id)) { return nil, fmt.Errorf("marketplace: invalid signing key ID %q", id) }
            key, err := base64.StdEncoding.DecodeString(value)
            if err != nil || len(key) != ed25519.PublicKeySize { return nil, fmt.Errorf("marketplace: signing key %q is not a valid Ed25519 public key", id) }
            keys[id] = ed25519.PublicKey(key)
        }
    }
    client := cfg.HTTPClient
    if client == nil {
        client = &http.Client{
            Timeout: 15 * time.Second,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                if len(via) >= 3 || req.URL.Scheme != "https" || len(via) == 0 || !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname()) {
                    return errors.New("marketplace: repository redirects must remain HTTPS on the original host")
                }
                return nil
            },
        }
    }
    return &Service{repo: repo, cfg: cfg, keys: keys, httpClient: client}, nil
}

func (s *Service) AddRepository(ctx context.Context, rawURL string) (*Source, error) {
    normalized, err := normalizeRepositoryURL(rawURL)
    if err != nil { return nil, err }
    digest := sha256.Sum256([]byte(normalized))
    id := hex.EncodeToString(digest[:10])
    existing, err := s.repo.GetSource(ctx, id)
    if err == nil { return existing, nil }
    if !errors.Is(err, ErrNotFound) { return nil, err }
    parsed, _ := url.Parse(normalized)
    now := time.Now().UTC()
    source := &Source{
        ID: id, URL: normalized, Name: parsed.Hostname(), Enabled: true,
        CreatedAt: now, UpdatedAt: now,
    }
    if err := s.repo.SaveSource(ctx, source); err != nil { return nil, err }
    return source, nil
}

func normalizeRepositoryURL(raw string) (string, error) {
    parsed, err := url.Parse(strings.TrimSpace(raw))
    if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
        return "", errors.New("marketplace: repository URL must be HTTPS and must not contain credentials, query or fragment")
    }
    if parsed.Path == "" || parsed.Path == "/" { parsed.Path = "" }
    parsed.Path = strings.TrimRight(parsed.Path, "/")
    return parsed.String(), nil
}

func (s *Service) ListRepositories(ctx context.Context) ([]*Source, error) {
    return s.repo.ListSources(ctx)
}

func (s *Service) RemoveRepository(ctx context.Context, id string) error {
    if _, err := s.repo.GetSource(ctx, id); err != nil { return err }
    packages, err := s.repo.ListPackages(ctx, id)
    if err != nil { return err }
    for _, pkg := range packages {
        if pkg.InstalledPath != "" {
            if _, err := s.UninstallArtifact(ctx, pkg.ID); err != nil {
                return fmt.Errorf("marketplace: cannot remove installed package %s: %w", pkg.ID, err)
            }
        }
    }
    return s.repo.DeleteSource(ctx, id)
}

func (s *Service) ListPackages(ctx context.Context, repositoryID string) ([]*Package, error) {
    return s.repo.ListPackages(ctx, repositoryID)
}

func (s *Service) SyncRepository(ctx context.Context, id string) (*Source, error) {
    source, err := s.repo.GetSource(ctx, id)
    if err != nil { return nil, err }
    if !source.Enabled { return nil, errors.New("marketplace: repository is disabled") }
    indexURL, err := url.JoinPath(source.URL, "index.json")
    if err != nil { return nil, err }
    body, err := s.fetch(ctx, indexURL, maxIndexBytes)
    if err != nil { return s.saveSyncFailure(ctx, source, err) }

    var index RepositoryIndex
    decoder := json.NewDecoder(bytes.NewReader(body))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&index); err != nil { return s.saveSyncFailure(ctx, source, fmt.Errorf("invalid index JSON: %w", err)) }
    if index.SchemaVersion != IndexSchemaVersion { return s.saveSyncFailure(ctx, source, fmt.Errorf("unsupported plugin index schema_version %d", index.SchemaVersion)) }
    if len(index.Packages) > maxPackagesPerIndex { return s.saveSyncFailure(ctx, source, errors.New("index contains too many packages")) }
    if strings.TrimSpace(index.Repository.ID) == "" || strings.TrimSpace(index.Repository.Name) == "" {
        return s.saveSyncFailure(ctx, source, errors.New("repository id and name are required"))
    }

    trusted, err := s.verifyIndexSignature(ctx, source.URL, index.Repository.KeyID, body)
    if err != nil { return s.saveSyncFailure(ctx, source, err) }
    if !trusted && !s.cfg.AllowUntrustedRepositories {
        return s.saveSyncFailure(ctx, source, ErrUntrusted)
    }

    now := time.Now().UTC()
    existingPackages, err := s.repo.ListPackages(ctx, id)
    if err != nil {
        _, syncErr := s.saveSyncFailure(ctx, source, err)
        return nil, syncErr
    }
    validated := make([]*Package, 0, len(index.Packages))
    packageIDs := make(map[string]struct{}, len(index.Packages))
    for _, entry := range index.Packages {
        if _, duplicate := packageIDs[entry.ID]; duplicate {
            return s.saveSyncFailure(ctx, source, fmt.Errorf("duplicate package ID %q in index", entry.ID))
        }
        pkg, err := s.validatePackage(source, index.Repository, entry, trusted, now)
        if err != nil { return s.saveSyncFailure(ctx, source, err) }
        packageIDs[pkg.ID] = struct{}{}
        if current, getErr := s.repo.GetPackage(ctx, pkg.ID); getErr == nil {
            if current.Version == pkg.Version && strings.EqualFold(current.Artifact.SHA256, pkg.Artifact.SHA256) {
                pkg.InstalledPath = current.InstalledPath
                pkg.InstalledAt = current.InstalledAt
            }
        } else if !errors.Is(getErr, ErrNotFound) {
            return s.saveSyncFailure(ctx, source, getErr)
        }
        validated = append(validated, pkg)
    }
    for _, pkg := range validated {
        if err := s.repo.SavePackage(ctx, pkg); err != nil { return s.saveSyncFailure(ctx, source, err) }
    }
    // Remove entries withdrawn from the index, including staged copies, so a
    // repository refresh cannot leave orphaned or silently stale extensions.
    for _, old := range existingPackages {
        if _, stillPublished := packageIDs[old.ID]; !stillPublished {
            if old.InstalledPath != "" {
                if _, err := s.UninstallArtifact(ctx, old.ID); err != nil {
                    return s.saveSyncFailure(ctx, source, fmt.Errorf("remove withdrawn package %s: %w", old.ID, err))
                }
            }
            if err := s.repo.DeletePackage(ctx, old.ID); err != nil {
                return s.saveSyncFailure(ctx, source, err)
            }
        }
    }

    hash := sha256.Sum256(body)
    source.Name = index.Repository.Name
    source.Owner = index.Repository.Owner
    source.Description = index.Repository.Description
    source.KeyID = index.Repository.KeyID
    source.Trusted = trusted
    source.IndexSHA256 = hex.EncodeToString(hash[:])
    source.LastSyncedAt = &now
    source.LastError = ""
    source.UpdatedAt = now
    if err := s.repo.SaveSource(ctx, source); err != nil { return nil, err }
    return source, nil
}

func (s *Service) saveSyncFailure(ctx context.Context, source *Source, failure error) (*Source, error) {
    source.LastError = failure.Error()
    source.UpdatedAt = time.Now().UTC()
    _ = s.repo.SaveSource(ctx, source)
    return nil, failure
}

func (s *Service) verifyIndexSignature(ctx context.Context, repositoryURL, keyID string, body []byte) (bool, error) {
    if keyID == "" { return false, nil }
    key, found := s.keys[keyID]
    if !found { return false, nil }
    signatureURL, err := url.JoinPath(repositoryURL, "index.sig")
    if err != nil { return false, err }
    signature, err := s.fetch(ctx, signatureURL, 4096)
    if err != nil {
        if strings.Contains(err.Error(), "HTTP 404") { return false, nil }
        return false, fmt.Errorf("marketplace: fetch repository signature: %w", err)
    }
    decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
    if err != nil || len(decoded) != ed25519.SignatureSize {
        return false, fmt.Errorf("%w: detached index.sig is not a valid Base64 Ed25519 signature", ErrSignature)
    }
    if !ed25519.Verify(key, body, decoded) { return false, ErrSignature }
    return true, nil
}

func (s *Service) validatePackage(source *Source, repo IndexRepository, entry IndexPackage, trusted bool, now time.Time) (*Package, error) {
    if !packageIDPattern.MatchString(entry.ID) { return nil, fmt.Errorf("marketplace: invalid package ID %q", entry.ID) }
    if strings.TrimSpace(entry.Name) == "" || len(entry.Name) > 128 { return nil, fmt.Errorf("marketplace: package %q has an invalid name", entry.ID) }
    if !semverPattern.MatchString(entry.Version) { return nil, fmt.Errorf("marketplace: package %q has invalid semantic version", entry.ID) }
    if entry.Type != "wasm_plugin" && entry.Type != "blueprint" && entry.Type != "widget" && entry.Type != "integration" {
        return nil, fmt.Errorf("marketplace: package %q has unsupported type %q", entry.ID, entry.Type)
    }
    if entry.Runtime != "wasi" && entry.Runtime != "data" && entry.Runtime != "builtin" {
        return nil, fmt.Errorf("marketplace: package %q runtime must be wasi, data or builtin", entry.ID)
    }
    if entry.Runtime == "builtin" {
        return nil, fmt.Errorf("marketplace: package %q is a built-in adapter and cannot be installed from an external repository", entry.ID)
    }
    if !sha256Pattern.MatchString(entry.Artifact.SHA256) || entry.Artifact.Size < 1 || entry.Artifact.Size > s.cfg.MaxArtifactBytes {
        return nil, fmt.Errorf("marketplace: package %q has invalid artifact size or SHA-256", entry.ID)
    }
    artifactURL, err := url.Parse(entry.Artifact.URL)
    repoURL, _ := url.Parse(source.URL)
    if err != nil || artifactURL.Scheme != "https" || artifactURL.Host == "" || artifactURL.User != nil ||
        !strings.EqualFold(artifactURL.Hostname(), repoURL.Hostname()) || artifactURL.Port() != repoURL.Port() {
        return nil, fmt.Errorf("marketplace: package %q artifact must be HTTPS on the same host as its repository index", entry.ID)
    }
    if entry.MinDomusVersion != "" && !semverPattern.MatchString(entry.MinDomusVersion) {
        return nil, fmt.Errorf("marketplace: package %q has an invalid min_domus_version", entry.ID)
    }
    if entry.MaxDomusVersion != "" && !semverPattern.MatchString(entry.MaxDomusVersion) {
        return nil, fmt.Errorf("marketplace: package %q has an invalid max_domus_version", entry.ID)
    }
    compatible := true
    if entry.MinDomusVersion != "" && compareVersions(s.cfg.DomusVersion, entry.MinDomusVersion) < 0 { compatible = false }
    if entry.MaxDomusVersion != "" && compareVersions(s.cfg.DomusVersion, entry.MaxDomusVersion) > 0 { compatible = false }
    return &Package{
        ID: entry.ID, RepositoryID: source.ID, Name: entry.Name, Type: entry.Type, Version: entry.Version,
        Author: entry.Author, Description: entry.Description, License: entry.License, Runtime: entry.Runtime,
        MinDomusVersion: entry.MinDomusVersion, MaxDomusVersion: entry.MaxDomusVersion,
        Capabilities: append([]string(nil), entry.Capabilities...), Artifact: entry.Artifact,
        Trusted: trusted, Compatible: compatible, UpdatedAt: now,
    }, nil
}

func compareVersions(a, b string) int {
    parse := func(value string) [3]int {
        value = strings.TrimPrefix(value, "v")
        value = strings.SplitN(strings.SplitN(value, "-", 2)[0], "+", 2)[0]
        var result [3]int
        parts := strings.Split(value, ".")
        for i := 0; i < len(parts) && i < 3; i++ { _, _ = fmt.Sscanf(parts[i], "%d", &result[i]) }
        return result
    }
    x, y := parse(a), parse(b)
    for i := 0; i < 3; i++ { if x[i] < y[i] { return -1 }; if x[i] > y[i] { return 1 } }
    return 0
}

func (s *Service) fetch(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
    parsed, err := url.Parse(rawURL)
    if err != nil || parsed.Scheme != "https" || parsed.Host == "" { return nil, errors.New("marketplace: fetch URL must use HTTPS") }
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
    if err != nil { return nil, err }
    req.Header.Set("Accept", "application/json, application/octet-stream, text/plain")
    resp, err := s.httpClient.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { return nil, fmt.Errorf("marketplace: HTTP %d fetching %s", resp.StatusCode, parsed.Redacted()) }
    if resp.ContentLength > limit { return nil, fmt.Errorf("marketplace: response exceeds %d bytes", limit) }
    body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
    if err != nil { return nil, err }
    if int64(len(body)) > limit { return nil, fmt.Errorf("marketplace: response exceeds %d bytes", limit) }
    return body, nil
}

// InstallArtifact verifies the SHA-256, then stages the file atomically. It never
// executes or loads third-party code into the Domus process.
func (s *Service) InstallArtifact(ctx context.Context, packageID string) (*Package, error) {
    pkg, err := s.repo.GetPackage(ctx, packageID)
    if err != nil { return nil, err }
    if !pkg.Compatible { return nil, errors.New("marketplace: package is not compatible with this Domus version") }
    if !pkg.Trusted && !s.cfg.AllowUntrustedRepositories { return nil, ErrUntrusted }
    source, err := s.repo.GetSource(ctx, pkg.RepositoryID)
    if err != nil { return nil, err }
    artifactURL, err := url.Parse(pkg.Artifact.URL)
    sourceURL, _ := url.Parse(source.URL)
    if err != nil || !strings.EqualFold(artifactURL.Hostname(), sourceURL.Hostname()) || artifactURL.Scheme != "https" {
        return nil, errors.New("marketplace: artifact URL is outside the repository origin")
    }
    raw, err := s.fetch(ctx, pkg.Artifact.URL, s.cfg.MaxArtifactBytes)
    if err != nil { return nil, err }
    if int64(len(raw)) != pkg.Artifact.Size { return nil, errors.New("marketplace: artifact byte size does not match the index") }
    digest := sha256.Sum256(raw)
    actual := hex.EncodeToString(digest[:])
    if !strings.EqualFold(actual, pkg.Artifact.SHA256) { return nil, ErrArtifactIntegrity }

    ext := ".bin"
    if pkg.Runtime == "wasi" { ext = ".wasm" }
    destinationDir := filepath.Join(s.cfg.InstallRoot, source.ID, pkg.ID, pkg.Version)
    if err := os.MkdirAll(destinationDir, 0o700); err != nil { return nil, fmt.Errorf("marketplace: create install directory: %w", err) }
    temp, err := os.CreateTemp(destinationDir, ".download-*")
    if err != nil { return nil, err }
    tempName := temp.Name()
    defer os.Remove(tempName)
    if err := temp.Chmod(0o600); err != nil { _ = temp.Close(); return nil, err }
    if _, err := temp.Write(raw); err != nil { _ = temp.Close(); return nil, err }
    if err := temp.Sync(); err != nil { _ = temp.Close(); return nil, err }
    if err := temp.Close(); err != nil { return nil, err }
    finalPath := filepath.Join(destinationDir, pkg.ID+ext)
    if err := os.Rename(tempName, finalPath); err != nil { return nil, err }

    now := time.Now().UTC()
    pkg.InstalledPath = finalPath
    pkg.InstalledAt = &now
    pkg.UpdatedAt = now
    if err := s.repo.SavePackage(ctx, pkg); err != nil { _ = os.Remove(finalPath); return nil, err }
    return pkg, nil
}

func (s *Service) UninstallArtifact(ctx context.Context, packageID string) (*Package, error) {
    pkg, err := s.repo.GetPackage(ctx, packageID)
    if err != nil { return nil, err }
    if pkg.InstalledPath != "" {
        root, err := filepath.Abs(s.cfg.InstallRoot)
        if err != nil { return nil, err }
        path, err := filepath.Abs(pkg.InstalledPath)
        if err != nil { return nil, err }
        rel, err := filepath.Rel(root, path)
        if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
            return nil, errors.New("marketplace: refusing to remove an artifact outside the install root")
        }
        if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) { return nil, err }
        _ = os.Remove(filepath.Dir(path))
    }
    pkg.InstalledPath = ""
    pkg.InstalledAt = nil
    pkg.UpdatedAt = time.Now().UTC()
    if err := s.repo.SavePackage(ctx, pkg); err != nil { return nil, err }
    return pkg, nil
}


// ConfigFromEnv reads marketplace trust and staging settings. Only public signing
// keys are accepted here; private signing keys belong in the repository CI system.
func ConfigFromEnv() Config {
    maxBytes := int64(16 << 20)
    if raw := strings.TrimSpace(os.Getenv("DOMUS_STORE_MAX_ARTIFACT_BYTES")); raw != "" {
        if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value > 0 {
            maxBytes = value
        }
    }
    allowUntrusted := false
    if raw := strings.TrimSpace(os.Getenv("DOMUS_STORE_ALLOW_UNTRUSTED_REPOSITORIES")); raw != "" {
        allowUntrusted, _ = strconv.ParseBool(raw)
    }
    root := strings.TrimSpace(os.Getenv("DOMUS_STORE_INSTALL_ROOT"))
    if root == "" { root = "./data/marketplace" }
    return Config{
        TrustedKeysJSON: strings.TrimSpace(os.Getenv("DOMUS_STORE_TRUSTED_KEYS_JSON")),
        AllowUntrustedRepositories: allowUntrusted,
        InstallRoot: root,
        MaxArtifactBytes: maxBytes,
        DomusVersion: strings.TrimSpace(os.Getenv("DOMUS_VERSION")),
    }
}
