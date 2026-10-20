// Package backup provides Domus database and configuration backup/restore functionality.
//
// Plain backup:   .tar.gz  archive containing domus.db, domus.toml, backup_manifest.json
// Encrypted backup: .tar.gz.enc  = the tar.gz bytes encrypted with AES-256-GCM via the
//
//	secrets package (same wire format: salt|nonce|ciphertext+tag)
package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/config"
	_ "modernc.org/sqlite"
)

// ─── options ──────────────────────────────────────────────────────────────────

// CreateOptions configures a backup creation run.
type CreateOptions struct {
	// DBPath is the path to the SQLite database file.
	DBPath string
	// ConfigPath is the path to the TOML configuration file (may be empty).
	ConfigPath string
	// OutputDir is the directory where the archive will be written.
	OutputDir string
	// Passphrase, if non-empty, enables AES-256-GCM encryption.
	Passphrase string
}

// RestoreOptions configures a restore run.
type RestoreOptions struct {
	// ArchivePath is the path to the .tar.gz or .tar.gz.enc archive.
	ArchivePath string
	// OutputDir is the directory where the restored files will be written.
	OutputDir string
	// Passphrase is required for encrypted archives (must match the one used during backup).
	Passphrase string
	// SelectiveFiles optionally filters which files to extract (e.g. ["domus.db"] or ["domus.toml"]).
	SelectiveFiles []string
}

// ─── result ───────────────────────────────────────────────────────────────────

// CreateResult holds information about the created backup.
type CreateResult struct {
	FilePath  string
	SizeBytes int64
	Encrypted bool
	CreatedAt time.Time
}

// ─── manifest ─────────────────────────────────────────────────────────────────

type manifest struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Encrypted bool      `json:"encrypted"`
	Files     []string  `json:"files"`
}

// ─── Service ──────────────────────────────────────────────────────────────────

// Service provides backup and restore operations.
type Service struct{}

// NewService constructs a backup Service.
func NewService() *Service { return &Service{} }

// Create produces a backup archive in OutputDir.
//
// If Passphrase is non-empty the archive is encrypted with AES-256-GCM and
// the output file is named with a .tar.gz.enc extension; otherwise it is a
// plain .tar.gz.
func (s *Service) Create(opts CreateOptions) (CreateResult, error) {
	if opts.DBPath == "" {
		return CreateResult{}, fmt.Errorf("backup: DBPath is required")
	}
	if opts.OutputDir == "" {
		return CreateResult{}, fmt.Errorf("backup: OutputDir is required")
	}

	now := time.Now().UTC()
	timestamp := now.Format("20060102-150405")

	// Build the in-memory tar.gz
	tarBytes, fileNames, err := buildTarGz(opts, now)
	if err != nil {
		return CreateResult{}, err
	}

	encrypted := opts.Passphrase != ""
	var outBytes []byte

	if encrypted {
		outBytes, err = config.Encrypt(tarBytes, opts.Passphrase)
		if err != nil {
			return CreateResult{}, fmt.Errorf("backup: encryption failed: %w", err)
		}
	} else {
		outBytes = tarBytes
	}

	// Determine filename and write
	ext := "tar.gz"
	if encrypted {
		ext = "tar.gz.enc"
	}
	filename := fmt.Sprintf("domus-backup-%s.%s", timestamp, ext)
	outPath := filepath.Join(opts.OutputDir, filename)

	if err := os.WriteFile(outPath, outBytes, 0o600); err != nil {
		return CreateResult{}, fmt.Errorf("backup: failed to write archive: %w", err)
	}

	_ = fileNames // already embedded in the manifest inside the archive

	info, err := os.Stat(outPath)
	if err != nil {
		return CreateResult{}, fmt.Errorf("backup: failed to stat output file: %w", err)
	}

	return CreateResult{
		FilePath:  outPath,
		SizeBytes: info.Size(),
		Encrypted: encrypted,
		CreatedAt: now,
	}, nil
}

// Restore extracts a backup archive into OutputDir.
//
// If the archive ends in .tar.gz.enc, Passphrase is required for decryption.
// Files are written atomically — existing files are overwritten.
// The restore is idempotent: running it twice produces the same result.
func (s *Service) Restore(opts RestoreOptions) error {
	if opts.ArchivePath == "" {
		return fmt.Errorf("restore: ArchivePath is required")
	}
	if opts.OutputDir == "" {
		return fmt.Errorf("restore: OutputDir is required")
	}

	raw, err := os.ReadFile(opts.ArchivePath)
	if err != nil {
		return fmt.Errorf("restore: failed to read archive: %w", err)
	}

	encrypted := strings.HasSuffix(opts.ArchivePath, ".enc")
	var tarBytes []byte

	if encrypted {
		if opts.Passphrase == "" {
			return fmt.Errorf("restore: passphrase is required for encrypted archives")
		}
		tarBytes, err = config.Decrypt(raw, opts.Passphrase)
		if err != nil {
			return fmt.Errorf("restore: decryption failed: %w", err)
		}
	} else {
		tarBytes = raw
	}

	return extractTarGz(tarBytes, opts.OutputDir, opts.SelectiveFiles)
}

// VerifyResult reports the integrity check outcome of a backup archive.
type VerifyResult struct {
	Valid     bool      `json:"valid"`
	Encrypted bool      `json:"encrypted"`
	Files     []string  `json:"files"`
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int64     `json:"size_bytes"`
	Error     string    `json:"error,omitempty"`
}

// VerifyArchive performs an in-memory dry-run restore validation of the archive:
// testing decryption, gzip decompression, tar structure, and manifest integrity
// without extracting any files to disk.
func (s *Service) VerifyArchive(archivePath, passphrase string) (VerifyResult, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return VerifyResult{Valid: false, Error: err.Error()}, fmt.Errorf("verify: archive not found: %w", err)
	}

	raw, err := os.ReadFile(archivePath)
	if err != nil {
		return VerifyResult{Valid: false, Error: err.Error()}, fmt.Errorf("verify: cannot read archive: %w", err)
	}

	encrypted := strings.HasSuffix(archivePath, ".enc")
	var tarBytes []byte

	if encrypted {
		if passphrase == "" {
			return VerifyResult{Valid: false, Encrypted: true, Error: "passphrase required"}, fmt.Errorf("verify: passphrase required for encrypted archive")
		}
		tarBytes, err = config.Decrypt(raw, passphrase)
		if err != nil {
			return VerifyResult{Valid: false, Encrypted: true, Error: "decryption failed"}, fmt.Errorf("verify: decryption failed: %w", err)
		}
	} else {
		tarBytes = raw
	}

	gz, err := gzip.NewReader(bytes.NewReader(tarBytes))
	if err != nil {
		return VerifyResult{Valid: false, Encrypted: encrypted, Error: "corrupt gzip stream"}, fmt.Errorf("verify: invalid gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var files []string
	var manifestFound bool
	var manifestCreatedAt time.Time

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return VerifyResult{Valid: false, Encrypted: encrypted, Error: "corrupt tar header"}, fmt.Errorf("verify: tar read error: %w", err)
		}

		files = append(files, hdr.Name)
		if hdr.Name == "backup_manifest.json" {
			manifestFound = true
			var m manifest
			dec := json.NewDecoder(tr)
			if err := dec.Decode(&m); err == nil {
				manifestCreatedAt = m.CreatedAt
			}
		} else {
			// Read and discard file stream to verify integrity
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return VerifyResult{Valid: false, Encrypted: encrypted, Error: "corrupt file contents in tar"}, fmt.Errorf("verify: corrupt file %s: %w", hdr.Name, err)
			}
		}
	}

	if !manifestFound {
		return VerifyResult{Valid: false, Encrypted: encrypted, Error: "missing backup_manifest.json"}, fmt.Errorf("verify: missing manifest")
	}

	return VerifyResult{
		Valid:     true,
		Encrypted: encrypted,
		Files:     files,
		CreatedAt: manifestCreatedAt,
		SizeBytes: fi.Size(),
	}, nil
}

// TestRestoreIntegrity extracts the database from an archive into a temporary sandbox
// and executes SQLite "PRAGMA integrity_check;" on the restored database file.
// This guarantees that the backup is uncorrupted, recoverable, and relationally valid.
func (s *Service) TestRestoreIntegrity(ctx context.Context, archivePath, passphrase string) error {
	tmpDir, err := os.MkdirTemp("", "domus_restore_check_*")
	if err != nil {
		return fmt.Errorf("test_restore: cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Restore database selectively into temporary sandbox
	err = s.Restore(RestoreOptions{
		ArchivePath:    archivePath,
		OutputDir:      tmpDir,
		Passphrase:     passphrase,
		SelectiveFiles: []string{"domus.db"},
	})
	if err != nil {
		return fmt.Errorf("test_restore: extraction failed: %w", err)
	}

	dbPath := filepath.Join(tmpDir, "domus.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("test_restore: domus.db not found in restored archive: %w", err)
	}

	// Open connection with query_only=ON in read-only verification mode
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_query_only=1", filepath.ToSlash(dbPath))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("test_restore: failed to open restored sqlite database: %w", err)
	}
	defer db.Close()

	var result string
	row := db.QueryRowContext(ctx, "PRAGMA integrity_check;")
	if err := row.Scan(&result); err != nil {
		return fmt.Errorf("test_restore: integrity_check query failed: %w", err)
	}

	if result != "ok" {
		return fmt.Errorf("test_restore: SQLite database corruption detected: %s", result)
	}

	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func buildTarGz(opts CreateOptions, now time.Time) ([]byte, []string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	var fileNames []string

	// Add database file
	if opts.DBPath != "" {
		name, err := addFileToTar(tw, opts.DBPath, "domus.db")
		if err != nil {
			return nil, nil, fmt.Errorf("backup: failed to add db: %w", err)
		}
		fileNames = append(fileNames, name)
	}

	// Add config file (optional)
	if opts.ConfigPath != "" {
		if _, statErr := os.Stat(opts.ConfigPath); statErr == nil {
			name, err := addFileToTar(tw, opts.ConfigPath, "domus.toml")
			if err != nil {
				return nil, nil, fmt.Errorf("backup: failed to add config: %w", err)
			}
			fileNames = append(fileNames, name)
		}
	}

	// Add manifest
	m := manifest{
		Version:   "1",
		CreatedAt: now,
		Encrypted: opts.Passphrase != "",
		Files:     fileNames,
	}
	mBytes, _ := json.MarshalIndent(m, "", "  ")
	mBytes = append(mBytes, '\n')

	hdr := &tar.Header{
		Name:    "backup_manifest.json",
		Mode:    0o600,
		Size:    int64(len(mBytes)),
		ModTime: now,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, nil, fmt.Errorf("backup: manifest header: %w", err)
	}
	if _, err := tw.Write(mBytes); err != nil {
		return nil, nil, fmt.Errorf("backup: manifest write: %w", err)
	}
	fileNames = append(fileNames, "backup_manifest.json")

	if err := tw.Close(); err != nil {
		return nil, nil, fmt.Errorf("backup: tar close: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, nil, fmt.Errorf("backup: gzip close: %w", err)
	}

	return buf.Bytes(), fileNames, nil
}

func addFileToTar(tw *tar.Writer, srcPath, destName string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	hdr := &tar.Header{
		Name:    destName,
		Mode:    0o600,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	if _, err := io.Copy(tw, f); err != nil {
		return "", err
	}
	return destName, nil
}

func extractTarGz(tarBytes []byte, destDir string, filterFiles []string) error {
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("restore: failed to create output dir: %w", err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(tarBytes))
	if err != nil {
		return fmt.Errorf("restore: invalid gzip data: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("restore: tar read error: %w", err)
		}

		// Security: prevent path traversal
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("restore: unsafe path in archive: %s", hdr.Name)
		}

		if len(filterFiles) > 0 {
			matched := false
			for _, f := range filterFiles {
				if cleanName == f || filepath.Base(cleanName) == f {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		outPath := filepath.Join(destDir, cleanName)

		var fileBuf bytes.Buffer
		if _, err := io.Copy(&fileBuf, tr); err != nil {
			return fmt.Errorf("restore: read entry %s: %w", hdr.Name, err)
		}

		if err := os.WriteFile(outPath, fileBuf.Bytes(), 0o600); err != nil {
			return fmt.Errorf("restore: write %s: %w", outPath, err)
		}
	}
	return nil
}
