package supervisor

import (
	"bufio"
	"context"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// MemoryInfo captures RAM usage details.
type MemoryInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	UsedPercent    float64 `json:"used_percent"`
}

// SystemDiagnostics aggregates hardware metrics, slot state, and runtime status.
type SystemDiagnostics struct {
	// Hardware Telemetry
	SoCTempC             float64 `json:"soc_temp_c"`
	CPULoad1m            float64 `json:"cpu_load_1m"`
	CPULoad5m            float64 `json:"cpu_load_5m"`
	CPULoad15m           float64 `json:"cpu_load_15m"`
	CPUCores             int     `json:"cpu_cores"`
	MemoryTotalBytes     uint64  `json:"memory_total_bytes"`
	MemoryAvailableBytes uint64  `json:"memory_available_bytes"`
	MemoryUsedBytes      uint64  `json:"memory_used_bytes"`
	MemoryUsedPercent    float64 `json:"memory_used_percent"`
	DiskTotalBytes       uint64  `json:"disk_total_bytes"`
	DiskFreeBytes        uint64  `json:"disk_free_bytes"`
	DiskUsedBytes        uint64  `json:"disk_used_bytes"`
	DiskUsedPercent      float64 `json:"disk_used_percent"`

	// Appliance Platform
	Hostname      string `json:"hostname"`
	OSVersion     string `json:"os_version"`
	KernelVersion string `json:"kernel_version"`
	Architecture  string `json:"architecture"`
	UptimeSeconds int64  `json:"uptime_seconds"`

	// A/B Dual Slot Updates
	ActiveSlot    string `json:"active_slot"`
	BootedSlot    string `json:"booted_slot"`
	SlotConfirmed bool   `json:"slot_confirmed"`
	RaucStatus    string `json:"rauc_status"`

	// Containers & Applications
	CoreHealthy       bool            `json:"core_healthy"`
	CoreStatus        ContainerStatus `json:"core_status"`
	TotalContainers   int             `json:"total_containers"`
	RunningContainers int             `json:"running_containers"`
	InstalledApps     int             `json:"installed_apps"`

	// Maintenance State
	MaintenanceMode bool `json:"maintenance_mode"`

	Timestamp time.Time `json:"timestamp"`
}

// DiagnosticExport holds a complete sanitized bundle for export/support.
type DiagnosticExport struct {
	ExportedAt  time.Time              `json:"exported_at"`
	Diagnostics SystemDiagnostics      `json:"diagnostics"`
	Containers  []ContainerStatus      `json:"containers"`
	Apps        []AppSummary           `json:"apps"`
	Environment map[string]string      `json:"environment"`
	SystemInfo  map[string]interface{} `json:"system_info"`
}

// DiagnosticsCollector gathers metrics and system state.
type DiagnosticsCollector struct {
	cfg            Config
	runtime        ContainerRuntime
	rauc           RaucController
	appMgr         *AppManager
	maintenanceMgr *MaintenanceManager

	thermalZonePath string
	meminfoPath     string
	loadavgPath     string
	dataDirPath     string
	osReleasePath   string
	startedAt       time.Time
}

// NewDiagnosticsCollector creates a collector with default OS paths.
func NewDiagnosticsCollector(
	cfg Config,
	runtime ContainerRuntime,
	rauc RaucController,
	appMgr *AppManager,
) *DiagnosticsCollector {
	return &DiagnosticsCollector{
		cfg:             cfg,
		runtime:         runtime,
		rauc:            rauc,
		appMgr:          appMgr,
		thermalZonePath: "/sys/class/thermal/thermal_zone0/temp",
		meminfoPath:     "/proc/meminfo",
		loadavgPath:     "/proc/loadavg",
		dataDirPath:     cfg.DataDir,
		osReleasePath:   "/etc/os-release",
		startedAt:       time.Now(),
	}
}

// SetMaintenanceManager registers the maintenance manager reference.
func (c *DiagnosticsCollector) SetMaintenanceManager(mgr *MaintenanceManager) {
	c.maintenanceMgr = mgr
}

// Collect reads hardware metrics, slots, containers, and maintenance state.
func (c *DiagnosticsCollector) Collect(ctx context.Context) (*SystemDiagnostics, error) {
	tempC := c.readSoCTemp()
	mem := c.readMemInfo()
	load1, load5, load15 := c.readCPULoad()
	osName, kernelVer := c.readOSInfo()

	diskTotal, diskFree, diskUsed, diskPercent, _ := getDiskUsage(c.dataDirPath)

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "domus"
	}

	diag := &SystemDiagnostics{
		SoCTempC:             tempC,
		CPULoad1m:            load1,
		CPULoad5m:            load5,
		CPULoad15m:           load15,
		CPUCores:             runtime.NumCPU(),
		MemoryTotalBytes:     mem.TotalBytes,
		MemoryAvailableBytes: mem.AvailableBytes,
		MemoryUsedBytes:      mem.UsedBytes,
		MemoryUsedPercent:    mem.UsedPercent,
		DiskTotalBytes:       diskTotal,
		DiskFreeBytes:        diskFree,
		DiskUsedBytes:        diskUsed,
		DiskUsedPercent:      diskPercent,
		Hostname:             hostname,
		OSVersion:            osName,
		KernelVersion:        kernelVer,
		Architecture:         runtime.GOARCH,
		UptimeSeconds:        int64(time.Since(c.startedAt).Seconds()),
		Timestamp:            time.Now().UTC(),
	}

	// Dual slot update telemetry
	if c.rauc != nil {
		if rStatus, err := c.rauc.GetStatus(ctx); err == nil {
			diag.ActiveSlot = rStatus.ActiveSlot
			diag.BootedSlot = rStatus.BootedSlot
			diag.RaucStatus = rStatus.Status
			diag.SlotConfirmed = (rStatus.Status == "good" || rStatus.Status == "active")
		}
	}

	// Container & Core status
	if c.runtime != nil {
		containers, err := c.runtime.ListContainers(ctx)
		if err == nil {
			diag.TotalContainers = len(containers)
			running := 0
			for _, ct := range containers {
				if ct.State == ContainerStateRunning {
					running++
				}
				if ct.Name == "domus-core" {
					diag.CoreStatus = ct
				}
			}
			diag.RunningContainers = running
		}
	}

	// Apps count
	if c.appMgr != nil {
		if apps, err := c.appMgr.ListApps(ctx); err == nil {
			diag.InstalledApps = len(apps)
		}
	}

	// Maintenance Mode state
	if c.maintenanceMgr != nil {
		diag.MaintenanceMode = c.maintenanceMgr.GetStatus().Active
	}

	return diag, nil
}

// Export produces a sanitized debug bundle suitable for sharing.
func (c *DiagnosticsCollector) Export(ctx context.Context) (*DiagnosticExport, error) {
	diag, err := c.Collect(ctx)
	if err != nil {
		return nil, err
	}

	var containers []ContainerStatus
	if c.runtime != nil {
		containers, _ = c.runtime.ListContainers(ctx)
	}

	var apps []AppSummary
	if c.appMgr != nil {
		apps, _ = c.appMgr.ListApps(ctx)
	}

	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	export := &DiagnosticExport{
		ExportedAt:  time.Now().UTC(),
		Diagnostics: *diag,
		Containers:  containers,
		Apps:        apps,
		Environment: sanitizeEnv(envMap),
		SystemInfo: map[string]interface{}{
			"go_version":    runtime.Version(),
			"num_goroutine": runtime.NumGoroutine(),
			"compiler":      runtime.Compiler,
		},
	}

	return export, nil
}

func (c *DiagnosticsCollector) readSoCTemp() float64 {
	data, err := os.ReadFile(c.thermalZonePath)
	if err != nil {
		return 0.0
	}
	str := strings.TrimSpace(string(data))
	val, err := strconv.ParseFloat(str, 64)
	if err != nil || val <= 0 {
		return 0.0
	}
	// SoC temp in millidegrees (e.g. 43500 -> 43.5)
	if val > 1000 {
		val = val / 1000.0
	}
	return math.Round(val*10) / 10
}

func (c *DiagnosticsCollector) readMemInfo() MemoryInfo {
	var info MemoryInfo
	file, err := os.Open(c.meminfoPath)
	if err != nil {
		return info
	}
	defer file.Close()

	var memTotal, memFree, memAvailable, buffers, cached uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}
		// /proc/meminfo reports values in kB
		bytesVal := val * 1024

		switch parts[0] {
		case "MemTotal:":
			memTotal = bytesVal
		case "MemFree:":
			memFree = bytesVal
		case "MemAvailable:":
			memAvailable = bytesVal
		case "Buffers:":
			buffers = bytesVal
		case "Cached:":
			cached = bytesVal
		}
	}

	info.TotalBytes = memTotal
	if memAvailable > 0 {
		info.AvailableBytes = memAvailable
	} else {
		info.AvailableBytes = memFree + buffers + cached
	}

	if info.TotalBytes > info.AvailableBytes {
		info.UsedBytes = info.TotalBytes - info.AvailableBytes
	}
	if info.TotalBytes > 0 {
		info.UsedPercent = math.Round((float64(info.UsedBytes)/float64(info.TotalBytes)*100)*10) / 10
	}

	return info
}

func (c *DiagnosticsCollector) readCPULoad() (float64, float64, float64) {
	data, err := os.ReadFile(c.loadavgPath)
	if err != nil {
		return 0.0, 0.0, 0.0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0.0, 0.0, 0.0
	}
	l1, _ := strconv.ParseFloat(fields[0], 64)
	l5, _ := strconv.ParseFloat(fields[1], 64)
	l15, _ := strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}

func (c *DiagnosticsCollector) readOSInfo() (osName string, kernelVer string) {
	osName = "Domus OS 1.0.0"
	if data, err := os.ReadFile(c.osReleasePath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				osName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
			}
		}
	}

	// Kernel version from /proc/sys/kernel/osrelease if available
	if kdata, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		kernelVer = strings.TrimSpace(string(kdata))
	} else {
		kernelVer = runtime.GOOS + "/" + runtime.GOARCH
	}

	return osName, kernelVer
}

func sanitizeEnv(env map[string]string) map[string]string {
	sanitized := make(map[string]string, len(env))
	sensitiveKeywords := []string{
		"PASS", "SECRET", "TOKEN", "KEY", "AUTH", "CREDENTIAL", "PRIVATE", "CERT", "COOKIE",
	}

	for k, v := range env {
		upperKey := strings.ToUpper(k)
		isSensitive := false
		for _, kw := range sensitiveKeywords {
			if strings.Contains(upperKey, kw) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}
