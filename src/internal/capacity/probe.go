package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/models"
)

// CapacityProbe holds the configuration and state for capacity monitoring
type CapacityProbe struct {
	config    *config.CapacityConfig
	db        *gorm.DB
	client    *asynq.Client
	scheduler *asynq.Scheduler
	cron      *cron.Cron
	logger    interface{} // Placeholder for logger
}

// CapacityConfig holds configuration for capacity monitoring
type CapacityConfig struct {
	Interval          time.Duration `mapstructure:"interval"`           // Default: 10 minutes
	ProbeCommand      string        `mapstructure:"probe_command"`      // Default: "df --output=pcent /melodee/storage"
	WarningThreshold  float64       `mapstructure:"warning_threshold"`  // Default: 80%
	AlertThreshold    float64       `mapstructure:"alert_threshold"`    // Default: 90%
	GraceIntervals    int           `mapstructure:"grace_intervals"`    // Default: 1 interval
	FailureMaxIntervals int         `mapstructure:"failure_max_intervals"` // Default: 2 intervals
}

// DefaultCapacityConfig returns the default capacity configuration
func DefaultCapacityConfig() *CapacityConfig {
	return &CapacityConfig{
		Interval:          10 * time.Minute,
		ProbeCommand:      "df --output=pcent /melodee/storage",
		WarningThreshold:  80.0,
		AlertThreshold:    90.0,
		GraceIntervals:    1,
		FailureMaxIntervals: 2,
	}
}

// NewCapacityProbe creates a new capacity probe instance
func NewCapacityProbe(
	config *config.CapacityConfig,
	db *gorm.DB,
	client *asynq.Client,
	scheduler *asynq.Scheduler,
	logger interface{},
) *CapacityProbe {
	if config == nil {
		config = DefaultCapacityConfig()
	}

	cp := &CapacityProbe{
		config:    config,
		db:        db,
		client:    client,
		scheduler: scheduler,
		logger:    logger,
		cron:      cron.New(),
	}

	return cp
}

// CapacityStatus represents the status of capacity monitoring
type CapacityStatus struct {
	Path          string    `json:"path"`
	UsedPercent   float64   `json:"used_percent"`
	Status        string    `json:"status"` // "ok", "warning", "alert", "unknown"
	LatestReadAt  time.Time `json:"latest_read_at"`
	ErrorCount    int       `json:"error_count"`
	LastError     string    `json:"last_error"`
	NextCheckAt   time.Time `json:"next_check_at"`
}

// CapacityProbeResult holds the result of a single probe
type CapacityProbeResult struct {
	Path        string  `json:"path"`
	UsedPercent float64 `json:"used_percent"`
	Error       error   `json:"error,omitempty"`
	Command     string  `json:"command"`
	Output      string  `json:"output"`
}

// Start begins capacity monitoring
func (cp *CapacityProbe) Start() error {
	// Schedule the capacity check using cron
	spec := fmt.Sprintf("@every %s", cp.config.Interval.String())
	_, err := cp.cron.AddFunc(spec, cp.runProbe)
	if err != nil {
		return fmt.Errorf("failed to schedule capacity probe: %w", err)
	}

	cp.cron.Start()
	return nil
}

// Stop stops capacity monitoring
func (cp *CapacityProbe) Stop() {
	if cp.cron != nil {
		cp.cron.Stop()
	}
}

// runProbe executes a single capacity probe cycle
func (cp *CapacityProbe) runProbe() {
	// Get all library paths that need monitoring
	libraries, err := cp.getLibrariesToMonitor()
	if err != nil {
		cp.logError("Failed to get libraries for monitoring: %v", err)
		return
	}

	for _, lib := range libraries {
		result := cp.probePath(lib.Path)
		
		// Update capacity status in database
		if err := cp.updateCapacityStatus(&lib, result); err != nil {
			cp.logError("Failed to update capacity status for path %s: %v", lib.Path, err)
		}
	}
}

// probePath executes the capacity probe command against a specific path
func (cp *CapacityProbe) probePath(path string) *CapacityProbeResult {
	result := &CapacityProbeResult{
		Path:    path,
		Command: strings.Replace(cp.config.ProbeCommand, "/melodee/storage", path, -1),
	}

	// Split command into args
	parts := strings.Fields(result.Command)
	if len(parts) == 0 {
		result.Error = fmt.Errorf("invalid command: %s", result.Command)
		return result
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Errorf("command failed: %v", err)
		result.Output = string(output)
		return result
	}

	result.Output = string(output)

	// Parse the output to extract percentage
	percent, err := cp.parsePercentageFromOutput(result.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse output: %v", err)
		return result
	}

	result.UsedPercent = percent
	return result
}

// parsePercentageFromOutput extracts the usage percentage from df command output
func (cp *CapacityProbe) parsePercentageFromOutput(output string) (float64, error) {
	// Example output: "Use%\n95%" or "Use%\n85%\n"
	lines := strings.Split(output, "\n")
	
	// Look for the percentage in the output
	percentRegex := regexp.MustCompile(`(\d+)%`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := percentRegex.FindStringSubmatch(line); len(matches) > 1 {
			percent, err := strconv.ParseFloat(matches[1], 64)
			if err != nil {
				continue
			}
			return percent, nil
		}
	}
	
	return 0, fmt.Errorf("could not find percentage in output: %s", output)
}

// getLibrariesToMonitor gets libraries that should be monitored for capacity
func (cp *CapacityProbe) getLibrariesToMonitor() ([]models.Library, error) {
	var libraries []models.Library
	err := cp.db.Where("type = ? AND is_locked = ?", "production", false).Find(&libraries).Error
	if err != nil {
		return nil, err
	}
	
	return libraries, nil
}

// updateCapacityStatus updates the capacity status in the database
func (cp *CapacityProbe) updateCapacityStatus(library *models.Library, result *CapacityProbeResult) error {
	// Get or create capacity status for this library path
	var capacityStatus models.CapacityStatus
	err := cp.db.Where("library_id = ?", library.ID).First(&capacityStatus).Error
	
	var shouldCreate bool
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			shouldCreate = true
			capacityStatus.LibraryID = library.ID
		} else {
			return err
		}
	}

	// Determine status based on thresholds
	status := "ok"
	if result.Error != nil {
		// Increment error counter
		if shouldCreate {
			capacityStatus.ErrorCount = 1
		} else {
			capacityStatus.ErrorCount++
		}
		
		// Check if we're past the grace period for errors
		if capacityStatus.ErrorCount > cp.config.FailureMaxIntervals {
			status = "unknown"
		} else {
			// During grace period, status remains as last known state
			if capacityStatus.Status != "" {
				status = capacityStatus.Status
			} else {
				status = "unknown"
			}
		}
	} else {
		// Reset error counter on successful probe
		capacityStatus.ErrorCount = 0
		
		// Determine status based on percentage
		if result.UsedPercent >= cp.config.AlertThreshold {
			status = "alert"
		} else if result.UsedPercent >= cp.config.WarningThreshold {
			status = "warning"
		}
	}

	// Update fields
	capacityStatus.Path = result.Path
	capacityStatus.UsedPercent = result.UsedPercent
	capacityStatus.Status = status
	capacityStatus.LatestReadAt = time.Now()
	capacityStatus.NextCheckAt = time.Now().Add(cp.config.Interval)
	
	if result.Error != nil {
		capacityStatus.LastError = result.Error.Error()
	} else {
		capacityStatus.LastError = ""
	}

	// Save to database
	if shouldCreate {
		err = cp.db.Create(&capacityStatus).Error
	} else {
		err = cp.db.Save(&capacityStatus).Error
	}
	
	if err != nil {
		return err
	}

	// If status is alert and above threshold, consider taking action
	if status == "alert" {
		cp.handleOverCapacity(capacityStatus)
	}

	return nil
}

// handleOverCapacity handles situations where capacity exceeds thresholds
func (cp *CapacityProbe) handleOverCapacity(status models.CapacityStatus) {
	cp.logInfo("Capacity threshold exceeded for library ID %d: %.2f%% used", 
		status.LibraryID, status.UsedPercent)
	
	// In a complete implementation, this might:
	// 1. Quarantine new uploads with reason "disk_full"
	// 2. Send alerts to monitoring systems
	// 3. Stop allocation of new space to this library
	// 4. Update system-wide capacity status
}

// GetCapacityStatusForLibrary returns the current capacity status for a specific library
func (cp *CapacityProbe) GetCapacityStatusForLibrary(libraryID int32) (*CapacityStatus, error) {
	var status models.CapacityStatus
	err := cp.db.Where("library_id = ?", libraryID).First(&status).Error
	if err != nil {
		return nil, err
	}

	return &CapacityStatus{
		Path:          status.Path,
		UsedPercent:   status.UsedPercent,
		Status:        status.Status,
		LatestReadAt:  status.LatestReadAt,
		ErrorCount:    status.ErrorCount,
		LastError:     status.LastError,
		NextCheckAt:   status.NextCheckAt,
	}, nil
}

// GetAllCapacityStatuses returns capacity status for all libraries
func (cp *CapacityProbe) GetAllCapacityStatuses() ([]CapacityStatus, error) {
	var statuses []models.CapacityStatus
	err := cp.db.Find(&statuses).Error
	if err != nil {
		return nil, err
	}

	// Convert to external format
	var results []CapacityStatus
	for _, status := range statuses {
		results = append(results, CapacityStatus{
			Path:          status.Path,
			UsedPercent:   status.UsedPercent,
			Status:        status.Status,
			LatestReadAt:  status.LatestReadAt,
			ErrorCount:    status.ErrorCount,
			LastError:     status.LastError,
			NextCheckAt:   status.NextCheckAt,
		})
	}

	return results, nil
}

// logInfo logs an informational message
func (cp *CapacityProbe) logInfo(format string, args ...interface{}) {
	if cp.logger != nil {
		// In a real implementation, use the actual logger
		fmt.Printf("INFO [CapacityProbe]: "+format+"\n", args...)
	}
}

// logError logs an error message
func (cp *CapacityProbe) logError(format string, args ...interface{}) {
	if cp.logger != nil {
		// In a real implementation, use the actual logger
		fmt.Printf("ERROR [CapacityProbe]: "+format+"\n", args...)
	}
}

// ProbeNow forces an immediate capacity probe
func (cp *CapacityProbe) ProbeNow() error {
	cp.runProbe()
	return nil
}