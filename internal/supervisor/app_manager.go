package supervisor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// AppSummary represents basic state for listing installed apps.
type AppSummary struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Version string         `json:"version"`
	State   ContainerState `json:"state"`
	Uptime  int64          `json:"uptime_seconds,omitempty"`
}

// AppManager handles registration, persistence, and isolation of Domus Apps.
type AppManager struct {
	mu          sync.RWMutex
	manifestDir string
	appsDir     string
	runtime     ContainerRuntime
	apps        map[string]*AppManifest
}

// NewAppManager initializes an AppManager with storage directory paths.
func NewAppManager(dataDir string, runtime ContainerRuntime) *AppManager {
	manifestDir := filepath.Join(dataDir, "supervisor", "manifests")
	appsDir := filepath.Join(dataDir, "apps")
	_ = os.MkdirAll(manifestDir, 0700)
	_ = os.MkdirAll(appsDir, 0700)

	mgr := &AppManager{
		manifestDir: manifestDir,
		appsDir:     appsDir,
		runtime:     runtime,
		apps:        make(map[string]*AppManifest),
	}
	_ = mgr.loadStoredManifests()
	return mgr
}

func (m *AppManager) loadStoredManifests() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.manifestDir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			data, err := os.ReadFile(filepath.Join(m.manifestDir, e.Name()))
			if err != nil {
				continue
			}
			manifest, err := ParseAndValidateManifest(data)
			if err == nil {
				m.apps[manifest.ID] = manifest
			}
		}
	}
	return nil
}

// Install validates the manifest, creates isolated directories, and configures the container.
func (m *AppManager) Install(ctx context.Context, manifestBytes []byte) (*AppManifest, error) {
	manifest, err := ParseAndValidateManifest(manifestBytes)
	if err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Create app isolated data root
	appStorageRoot := filepath.Join(m.appsDir, manifest.ID)
	if err := os.MkdirAll(appStorageRoot, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory for app %s: %w", manifest.ID, err)
	}

	// 2. Prepare isolated bind mounts
	var containerMounts []MountConfig
	for _, vol := range manifest.Permissions.Storage {
		volDir := filepath.Join(appStorageRoot, vol.VolumeName)
		if info, statErr := os.Lstat(volDir); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("storage volume %q must not be a symbolic link", vol.VolumeName)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("failed to inspect storage volume %q: %w", vol.VolumeName, statErr)
		}
		if err := os.MkdirAll(volDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create storage volume %q: %w", vol.VolumeName, err)
		}
		containerMounts = append(containerMounts, MountConfig{
			Source:   volDir,
			Target:   vol.Path,
			ReadOnly: false,
		})
	}

	// 3. Configure container
	containerName := "domus-app-" + manifest.ID
	cfg := ContainerConfig{
		Image:         manifest.Image,
		Command:       manifest.Entrypoint,
		Environment:   manifest.Environment,
		Mounts:        containerMounts,
		NetworkMode:   manifest.Permissions.Network.Mode,
		MemoryLimitMB: manifest.Resources.MemoryLimitMB,
		CPULimit:      manifest.Resources.CPULimit,
		ReadOnlyRoot:  true,
	}

	// Persist the manifest with owner-only permissions before registering the
	// container. Environment entries may contain credentials.
	manifestFile := filepath.Join(m.manifestDir, manifest.ID+".yaml")
	outData, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to encode app manifest: %w", err)
	}
	tmp, err := os.CreateTemp(m.manifestDir, ".manifest-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("failed to secure manifest file: %w", err)
	}
	if _, err := tmp.Write(outData); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("failed to write manifest file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("failed to sync manifest file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("failed to close manifest file: %w", err)
	}
	if err := os.Rename(tmpName, manifestFile); err != nil {
		return nil, fmt.Errorf("failed to persist manifest: %w", err)
	}

	if err := m.runtime.EnsureContainer(ctx, containerName, cfg); err != nil {
		_ = os.Remove(manifestFile)
		return nil, fmt.Errorf("failed to register container with runtime: %w", err)
	}

	m.apps[manifest.ID] = manifest
	return manifest, nil
}

// Start launches the app container task.
func (m *AppManager) Start(ctx context.Context, id string) error {
	m.mu.RLock()
	_, exists := m.apps[id]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("app %s not found", id)
	}
	return m.runtime.StartContainer(ctx, "domus-app-"+id)
}

// Stop gracefully terminates the app container.
func (m *AppManager) Stop(ctx context.Context, id string) error {
	m.mu.RLock()
	_, exists := m.apps[id]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("app %s not found", id)
	}
	return m.runtime.StopContainer(ctx, "domus-app-"+id, 10*time.Second)
}

// Uninstall stops the container and removes registration.
func (m *AppManager) Uninstall(ctx context.Context, id string, purgeData bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	manifest, exists := m.apps[id]
	if !exists {
		return fmt.Errorf("app %s not found", id)
	}

	// Stop container
	_ = m.runtime.StopContainer(ctx, "domus-app-"+id, 5*time.Second)

	// Remove manifest file
	_ = os.Remove(filepath.Join(m.manifestDir, id+".yaml"))
	delete(m.apps, id)

	if purgeData {
		_ = os.RemoveAll(filepath.Join(m.appsDir, id))
	}
	_ = manifest
	return nil
}

// GetApp returns the manifest and runtime container status for an app.
func (m *AppManager) GetApp(ctx context.Context, id string) (*AppManifest, ContainerStatus, error) {
	m.mu.RLock()
	manifest, exists := m.apps[id]
	m.mu.RUnlock()
	if !exists {
		return nil, ContainerStatus{}, fmt.Errorf("app %s not found", id)
	}

	status, _ := m.runtime.GetContainerStatus(ctx, "domus-app-"+id)
	return manifest, status, nil
}

// ListApps returns summary statuses of all installed apps.
func (m *AppManager) ListApps(ctx context.Context) ([]AppSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []AppSummary
	for id, app := range m.apps {
		st, _ := m.runtime.GetContainerStatus(ctx, "domus-app-"+id)
		summaries = append(summaries, AppSummary{
			ID:      id,
			Name:    app.Name,
			Version: app.Version,
			State:   st.State,
		})
	}
	return summaries, nil
}
