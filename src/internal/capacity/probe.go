package capacity

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// CapacityProbe provides cross-platform capacity checking
type CapacityProbe struct {
	logger interface{} // Placeholder for logger interface
}

// NewCapacityProbe creates a new capacity probe
func NewCapacityProbe(logger interface{}) *CapacityProbe {
	return &CapacityProbe{
		logger: logger,
	}
}

// ProbeResult represents the result of a capacity probe
type ProbeResult struct {
	Path        string  `json:"path"`
	TotalBytes  int64   `json:"total_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	FreeBytes   int64   `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
	Status      string  `json:"status"` // "ok", "warning", "error"
	Threshold   float64 `json:"threshold"`
	Error       string  `json:"error,omitempty"`
}

// ProbePath checks the capacity of a given path
func (p *CapacityProbe) ProbePath(path string) (*ProbeResult, error) {
	var result *ProbeResult
	var err error

	switch runtime.GOOS {
	case "linux", "darwin":
		result, err = p.probeUnix(path)
	case "windows":
		result, err = p.probeWindows(path)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err != nil {
		// Return a result with error status
		return &ProbeResult{
			Path:   path,
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	// Set threshold (80% warning, 90% alert as per spec)
	if result.UsedPercent >= 90.0 {
		result.Status = "alert"
		result.Threshold = 90.0
	} else if result.UsedPercent >= 80.0 {
		result.Status = "warning"
		result.Threshold = 80.0
	} else {
		result.Status = "ok"
		result.Threshold = 0.0
	}

	return result, nil
}

// probeUnix checks capacity on Unix-like systems (Linux, macOS) using df command
func (p *CapacityProbe) probeUnix(path string) (*ProbeResult, error) {
	cmd := exec.Command("df", "--output=size,used,avail", "-B1", path)
	output, err := cmd.Output()
	if err != nil {
		// Try alternative format that's compatible with more Unix systems
		cmd = exec.Command("df", "-k", path) // Use kilobytes as base unit
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("df command failed: %w", err)
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output format: %q", string(output))
	}

	// Parse the usage line (second line)
	usageLine := lines[1]
	parts := strings.Fields(usageLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected df output format: not enough fields: %q", usageLine)
	}

	// Parse sizes from df output (in bytes due to -B1 flag)
	totalBytes, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total bytes: %w", err)
	}

	usedBytes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse used bytes: %w", err)
	}

	freeBytes, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse free bytes: %w", err)
	}

	// Calculate used percentage
	var usedPercent float64
	if totalBytes > 0 {
		usedPercent = float64(usedBytes) / float64(totalBytes) * 100.0
	}

	result := &ProbeResult{
		Path:        path,
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}

	return result, nil
}

// probeWindows checks capacity on Windows systems using syscall
func (p *CapacityProbe) probeWindows(path string) (*ProbeResult, error) {
	// Get volume information using Windows API
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	// Convert path to the root directory format required by GetDiskFreeSpaceEx
	volumePath := getVolumePath(path)

	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(volumePath),
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk information: %w", err)
	}

	totalBytes := int64(totalNumberOfBytes)
	usedBytes := totalBytes - int64(totalNumberOfFreeBytes)
	freeBytes := int64(freeBytesAvailable)

	// Calculate used percentage
	var usedPercent float64
	if totalBytes > 0 {
		usedPercent = float64(usedBytes) / float64(totalBytes) * 100.0
	}

	result := &ProbeResult{
		Path:        path,
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}

	return result, nil
}

// getVolumePath converts a path to a volume path format required by Windows API
func getVolumePath(path string) string {
	// Convert to absolute path if it's relative
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't get the absolute path, return the original path
		// This is safer than failing completely
		return path
	}

	// For Windows, we typically need the root drive letter format (e.g., "C:\\")
	if len(absPath) >= 3 && absPath[1] == ':' && (absPath[2] == '\\' || absPath[2] == '/') {
		// This appears to be a drive letter path, return the drive root
		return absPath[:3] // "C:\"
	}

	// For UNC paths (network drives), we'll return as-is
	return absPath
}

// ProbePathSyscall uses system calls instead of external commands (cross-platform)
func (p *CapacityProbe) ProbePathSyscall(path string) (*ProbeResult, error) {
	var stat syscall.Statfs_t
	
	// Perform system call to get filesystem statistics
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("statfs failed for path %s: %w", path, err)
	}

	// Calculate capacities
	totalBytes := int64(stat.Blocks) * int64(stat.Bsize)
	freeBytes := int64(stat.Bfree) * int64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	// Calculate used percentage
	var usedPercent float64
	if totalBytes > 0 {
		usedPercent = float64(usedBytes) / float64(totalBytes) * 100.0
	}

	result := &ProbeResult{
		Path:        path,
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}

	// Set status based on thresholds
	if result.UsedPercent >= 90.0 {
		result.Status = "alert"
		result.Threshold = 90.0
	} else if result.UsedPercent >= 80.0 {
		result.Status = "warning"
		result.Threshold = 80.0
	} else {
		result.Status = "ok"
		result.Threshold = 0.0
	}

	return result, nil
}

// ProbePathAlternative provides an alternative implementation using os.Stat
func (p *CapacityProbe) ProbePathAlternative(path string) (*ProbeResult, error) {
	// Get file info for the path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	// Make sure it's a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", path)
	}

	// Use syscall.Statfs to get disk usage
	var stat syscall.Statfs_t
	
	// Perform system call to get filesystem statistics
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("statfs failed for path %s: %w", path, err)
	}

	// Calculate capacities
	totalBytes := int64(stat.Blocks) * int64(stat.Bsize)
	freeBytes := int64(stat.Bfree) * int64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	// Calculate used percentage
	var usedPercent float64
	if totalBytes > 0 {
		usedPercent = float64(usedBytes) / float64(totalBytes) * 100.0
	}

	// Set status based on thresholds (80% warning, 90% alert per spec)
	var status string
	var threshold float64
	if usedPercent >= 90.0 {
		status = "alert"
		threshold = 90.0
	} else if usedPercent >= 80.0 {
		status = "warning"
		threshold = 80.0
	} else {
		status = "ok"
		threshold = 0.0
	}

	result := &ProbeResult{
		Path:        path,
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
		Status:      status,
		Threshold:   threshold,
	}

	return result, nil
}

// MonitorPaths continuously monitors multiple paths for capacity
func (p *CapacityProbe) MonitorPaths(paths []string, interval time.Duration, callback func(*ProbeResult)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, path := range paths {
			result, err := p.ProbePath(path)
			if err != nil {
				// Create error result and call callback
				callback(&ProbeResult{
					Path:   path,
					Status: "error",
					Error:  err.Error(),
				})
			} else {
				callback(result)
			}
		}
	}
}

// GetRecommendedAction returns the recommended action based on status
func (r *ProbeResult) GetRecommendedAction() string {
	switch r.Status {
	case "alert":
		return "immediate action required - capacity near limit"
	case "warning":
		return "consider adding more storage or cleaning up"
	case "ok":
		return "capacity adequate"
	default:
		return "unknown status"
	}
}