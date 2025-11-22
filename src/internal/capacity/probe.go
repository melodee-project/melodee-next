package capacity

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"melodee/internal/metrics"
)

// CapacityProbe defines the interface for capacity probing
type CapacityProbe interface {
	GetUsage(path string) (UsageInfo, error)
	RegisterMetrics(metrics *metrics.Metrics)
}

// UsageInfo holds information about disk usage
type UsageInfo struct {
	Path        string
	Total       uint64 // Total bytes
	Used        uint64 // Used bytes
	Free        uint64 // Free bytes
	UsedPercent float64 // Percentage used (0-100)
	Thresholds  Thresholds
	Status      string // "ok", "warning", "alert", "unknown"
	Timestamp   time.Time
}

// Thresholds defines warning and alert thresholds
type Thresholds struct {
	WarnPercent  float64 // 80% by default according to CAPACITY_PROBES.md
	AlertPercent float64 // 90% by default according to CAPACITY_PROBES.md
}

// DefaultThresholds returns the default thresholds as defined in CAPACITY_PROBES.md
func DefaultThresholds() Thresholds {
	return Thresholds{
		WarnPercent:  80.0,
		AlertPercent: 90.0,
	}
}

// PlatformCapacityProbe implements cross-platform capacity probing
type PlatformCapacityProbe struct {
	metrics *metrics.Metrics
}

// NewPlatformCapacityProbe creates a new capacity probe
func NewPlatformCapacityProbe(metrics *metrics.Metrics) *PlatformCapacityProbe {
	return &PlatformCapacityProbe{
		metrics: metrics,
	}
}

// GetUsage retrieves usage information for a given path on the current platform
func (p *PlatformCapacityProbe) GetUsage(path string) (UsageInfo, error) {
	usageStat, err := disk.Usage(path)
	if err != nil {
		return UsageInfo{}, fmt.Errorf("failed to get disk usage for path %s: %w", path, err)
	}

	thresholds := DefaultThresholds()
	status := p.evaluateStatus(usageStat.UsedPercent, thresholds)

	usageInfo := UsageInfo{
		Path:        path,
		Total:       usageStat.Total,
		Used:        usageStat.Used,
		Free:        usageStat.Free,
		UsedPercent: usageStat.UsedPercent,
		Thresholds:  thresholds,
		Status:      status,
		Timestamp:   time.Now(),
	}

	// Record in metrics if available
	if p.metrics != nil {
		p.metrics.SetCapacityPercent(path, usageStat.UsedPercent)
	}

	return usageInfo, nil
}

// evaluateStatus evaluates the status based on usage percentage and thresholds
func (p *PlatformCapacityProbe) evaluateStatus(usedPercent float64, thresholds Thresholds) string {
	if usedPercent >= thresholds.AlertPercent {
		return "alert"
	} else if usedPercent >= thresholds.WarnPercent {
		return "warning"
	}
	return "ok"
}

// RegisterMetrics registers capacity-related metrics
func (p *PlatformCapacityProbe) RegisterMetrics(metrics *metrics.Metrics) {
	// Metrics are already handled in GetUsage method
	// This method exists to satisfy interface and for future expansion
}

// MonitorCapacity continuously monitors capacity at specified intervals
func (p *PlatformCapacityProbe) MonitorCapacity(path string, interval time.Duration, callback func(UsageInfo, error)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			usage, err := p.GetUsage(path)
			callback(usage, err)
		}
	}()
}

// ValidatePath checks if the path is valid for capacity monitoring
func (p *PlatformCapacityProbe) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	_, err := disk.Usage(path)
	if err != nil {
		return fmt.Errorf("path does not exist or is not accessible: %s", err)
	}

	return nil
}

// GetAllPathsUsage gets usage information for multiple paths
func (p *PlatformCapacityProbe) GetAllPathsUsage(paths []string) (map[string]UsageInfo, []error) {
	results := make(map[string]UsageInfo)
	var errors []error

	for _, path := range paths {
		usage, err := p.GetUsage(path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to get usage for %s: %w", path, err))
			continue
		}
		results[path] = usage
	}

	return results, errors
}

// IsAlerting returns true if any monitored path is over the alert threshold
func (p *PlatformCapacityProbe) IsAlerting(usages map[string]UsageInfo) bool {
	for _, usage := range usages {
		if usage.UsedPercent >= usage.Thresholds.AlertPercent {
			return true
		}
	}

	return false
}

// IsWarning returns true if any monitored path is over the warning threshold but below alert
func (p *PlatformCapacityProbe) IsWarning(usages map[string]UsageInfo) bool {
	for _, usage := range usages {
		if usage.UsedPercent >= usage.Thresholds.WarnPercent && usage.UsedPercent < usage.Thresholds.AlertPercent {
			return true
		}
	}

	return false
}