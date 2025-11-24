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
- `melodee_db_query_duration_seconds_bucket`: Database query performance histograms (histogram)
- `http_request_duration_seconds_bucket{handler, le}`: HTTP request duration metrics (histogram)
- `http_requests_total{method, handler, status}`: Total HTTP requests counter
- `melodee_dlq_size`: Size of dead letter queue
- `melodee_queue_size`: Size of job queues
- Exposed via `/metrics` endpoint on both Melodee API and OpenSubsonic servers
- Used for alerting in `monitoring/prometheus/alerts.yml`:
  - `CapacityWarning`: Triggered when `melodee_capacity_percent > 80` for 5m
  - `CapacityCritical`: Triggered when `melodee_capacity_percent > 90` for 2m
  - `HighAPIErrorRate`: Triggered when more than 10% of Melodee API requests return 5xx errors
  - `SlowAPIResponse`: Triggered when 95th percentile of Melodee API requests exceed 5 seconds
  - `HighOpenSubsonicErrorRate`: Triggered when more than 10% of OpenSubsonic API requests return 5xx errors
  - `SlowOpenSubsonicResponse`: Triggered when 95th percentile of OpenSubsonic API requests exceed 5 seconds
