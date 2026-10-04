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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/config"
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

	return extractTarGz(tarBytes, opts.OutputDir)
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

func extractTarGz(tarBytes []byte, destDir string) error {
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
