package backup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DestinationType specifies the offsite storage backend.
type DestinationType string

const (
	DestLocalPath DestinationType = "local_path" // NAS mount, SMB, NFS directory
	DestWebDAV    DestinationType = "webdav"     // Nextcloud, ownCloud, WebDAV server
	DestS3        DestinationType = "s3"         // AWS S3, Cloudflare R2, MinIO
)

// OffsiteDestination configures remote storage targets.
type OffsiteDestination struct {
	Type     DestinationType `json:"type"`
	Endpoint string          `json:"endpoint"` // e.g. "https://nextcloud.local/remote.php/dav/files/user/" or S3 endpoint
	Path     string          `json:"path"`     // e.g. "/backups" or bucket path
	Bucket   string          `json:"bucket"`   // for S3
	Username string          `json:"username,omitempty"`
	Password string          `json:"password,omitempty"`
	Token    string          `json:"token,omitempty"`
}

// ExportOffsite transfers a created backup archive to an offsite target.
func (s *Service) ExportOffsite(ctx context.Context, result CreateResult, dest OffsiteDestination) error {
	fileData, err := os.ReadFile(result.FilePath)
	if err != nil {
		return fmt.Errorf("offsite: failed to read backup file %s: %w", result.FilePath, err)
	}

	fileName := filepath.Base(result.FilePath)

	switch dest.Type {
	case DestLocalPath:
		targetDir := dest.Path
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("offsite: cannot create local target dir %s: %w", targetDir, err)
		}
		targetPath := filepath.Join(targetDir, fileName)
		return os.WriteFile(targetPath, fileData, 0644)

	case DestWebDAV:
		url := strings.TrimRight(dest.Endpoint, "/") + "/" + strings.TrimPrefix(dest.Path, "/")
		url = strings.TrimRight(url, "/") + "/" + fileName

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(string(fileData)))
		if err != nil {
			return fmt.Errorf("offsite: invalid webdav request: %w", err)
		}

		if dest.Username != "" || dest.Password != "" {
			req.SetBasicAuth(dest.Username, dest.Password)
		}
		if dest.Token != "" {
			req.Header.Set("Authorization", "Bearer "+dest.Token)
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("offsite: webdav upload failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("offsite: webdav server returned status %d: %s", resp.StatusCode, string(body))
		}
		return nil

	case DestS3:
		// Standard S3 REST PUT object
		url := dest.Endpoint
		if dest.Bucket != "" {
			url = strings.TrimRight(url, "/") + "/" + dest.Bucket
		}
		if dest.Path != "" {
			url = strings.TrimRight(url, "/") + "/" + strings.Trim(dest.Path, "/")
		}
		url = strings.TrimRight(url, "/") + "/" + fileName

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(string(fileData)))
		if err != nil {
			return fmt.Errorf("offsite: invalid S3 request: %w", err)
		}
		if dest.Token != "" {
			req.Header.Set("Authorization", "Bearer "+dest.Token)
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("offsite: s3 upload failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("offsite: s3 returned status %d", resp.StatusCode)
		}
		return nil

	default:
		return fmt.Errorf("offsite: unsupported destination type %s", dest.Type)
	}
}
