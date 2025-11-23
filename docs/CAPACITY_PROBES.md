# Library Capacity Probes

Purpose: define how production library capacity is checked and how failures are handled.

## Default Probe
- Command: `df --output=pcent /melodee/storage`
- Interval: 10m
- Thresholds: warn at 80%, stop allocations at 90% (quarantine reason `disk_full`).

## Cross-Platform Guidance
- Linux: use `df`; fallback to `stat -f` if `df` unavailable.
- macOS: `df -k /path` equivalent acceptable.
- Windows: use Go stdlib `syscall.Statfs` equivalent (or PowerShell `Get-PSDrive`) inside worker/agent; surface same percentages.

## Error Handling
- If probe command fails or returns unparsable output:
  - Mark capacity status `unknown`, log error, and allow operations for 1 grace interval.
  - If failure persists for 2 intervals, stop allocations and quarantine with reason `capacity_probe_failed`.
- Metrics: `melodee_capacity_percent{path}` gauge; `melodee_capacity_probe_failures_total`.

## Prometheus Metrics
- `melodee_capacity_percent{path}`: Percentage of capacity used for each storage path
- Exposed via `/metrics` endpoint on both Melodee API and OpenSubsonic servers
- Used for alerting in `monitoring/prometheus/alerts.yml`:
  - `CapacityWarning`: Triggered when `melodee_capacity_percent > 80` for 5m
  - `CapacityCritical`: Triggered when `melodee_capacity_percent > 90` for 2m
