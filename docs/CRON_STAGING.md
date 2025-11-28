# Staging Scan Job Design (Draft)

> This is a proposal document for how to implement a cron-driven staging workflow inside the Melodee worker. You can edit / trim this as you like; nothing here is code yet.

---

## Goals

- **Automated ingestion**: Periodically scan an inbound directory and process results into the staging area.
- **In-process, no shelling out**: Use the same Go packages as the `scan-inbound` and `process-scan` binaries, but call them directly from the worker.
- **DRY**: Do not duplicate business logic from `internal/scanner` or `internal/processor`. Any orchestration lives in a shared helper, not copied into multiple entrypoints.
- **Config-driven**: All paths, worker counts, and schedule settings come from configuration, not hard-coded constants.
- **Safe & observable**: Clear logging, optional dry-run mode, and guardrails (e.g., disabled by default).

---

## High-Level Architecture

1. **Shared Orchestration Helper (New)**
   - A single function that encapsulates: “run one end-to-end staging cycle: scan inbound → process scan DB → upsert `staging_items` in Postgres.”
   - Lives in a non-`main` package (e.g. `internal/workflow`, `internal/media`, or `internal/processing`).
   - Accepts configuration (paths, workers, flags) and dependencies (e.g., `*gorm.DB`, logger).
   - Uses existing packages:
     - `internal/scanner` for creating/opening scan DBs and scanning directories.
     - `internal/processor` for moving files into staging and creating `staging_items` via `StagingRepository`.

2. **Worker Scheduler (New)**
   - The existing worker (`src/worker/main.go`) already runs an Asynq server and has access to Redis, Postgres, and the app config.
   - A new scheduler component will periodically trigger the shared orchestration helper.
   - Two possible scheduling strategies:
     - **Simple internal scheduler** (e.g., `time.Ticker`, or a cron library like `robfig/cron` if you prefer cron expressions).
     - **Asynq-based scheduling** (enqueue a `TypeStagingScan` task at intervals; handler calls the helper).

3. **Configuration**
  - This should use the system configuration that is editable by the admin in the Admin UI.
  | Key | Value | Description |
  | :------- | :------: | -------: |
  | staging_scan.enabled | false | if enabled or not |
  | staging_scan.dry_run | false | if true then dry run only |
  | staging_scan.schedule | "0 */1 * * *" | cron for every hour |
  | staging_scan.workers  | 4  | number of worker goroutines |
  | staging_scan.rate_limit | 0  | 0 = unlimited |
  | staging_scan.scan_db_data_path | /var/melodee/scan-db | directory where temporary scan DB files are written |

  - The worker reads this section when a cron run starts. If `enabled` is `false`, the cron logic never runs.
  - The worker should use the `inbound` type library path as the source directory to scan.
  - The worker should use the `staging` type library path as the staging root when configuring `processor.ProcessorConfig.StagingRoot`.
  - Scan database files are written to a configured scan DB data path (e.g. `staging_scan.scan_db_data_path`) or another fixed location that is **not** any library path. This keeps temporary scan DB artifacts separate from both inbound and staging library trees.
  - These keys are stored in the application settings (editable via `/admin/settings`) and read by the worker at the start of a scan run.
---

## Shared Helper: Single Staging Cycle

**Intent:** A single Go function invoked on a schedule, that does what a human would do by manually running:

1. `./scan-inbound -path <inbound> -output <scans>`
2. `./process-scan -scan <generated-scan.db> -staging <staging>`

but **without** shelling out, and using the same underlying packages.

### Proposed Shape

- Package: `internal/workflow` (placeholder name; change as you like).
- File: e.g. `staging_job.go`.

```go
// StagingJobConfig holds all knobs for one run.
type StagingJobConfig struct {
    Workers   int
    RateLimit int
    DryRun    bool
}

// RunStagingJobCycle runs a full cycle: scan inbound → process to staging → write staging_items.
// It is the canonical implementation used by cron and any other callers.
func RunStagingJobCycle(ctx context.Context, cfg StagingJobConfig, db *gorm.DB, logger Logger) error {
  // 0) Resolve current runtime configuration and libraries
  //    - Read latest staging_scan.* settings (enabled, dry_run, workers, rate_limit, schedule, scan_output_dir)
  //    - Resolve inbound and staging libraries from the database configuration (type = "inbound" and type = "staging")
  //    - If either library is missing or multiple of a type exist, log an error and fail this run

    // 1) Create scan DB (like scan-inbound)
  //    - Resolve inbound library path (type = "inbound") from configured libraries
  //    - Determine scan DB output directory from configuration
  //    - scanner.NewScanDB(scanOutputDir)
    //    - scanner.NewFileScanner(scanDB, cfg.Workers)
  //    - fileScanner.ScanDirectory(inboundPath)
    //    - scanDB.ComputeAlbumGrouping()

    // 2) Process albums to staging (like process-scan)
    //    - Resolve staging library path (type = "staging") from configured libraries
    //    - proc := processor.NewProcessor(&processor.ProcessorConfig{StagingRoot: stagingPath, Workers: cfg.Workers, RateLimit: cfg.RateLimit, DryRun: cfg.DryRun}, scanDB)
    //    - results, err := proc.ProcessAllAlbums()

    // 3) Persist to Postgres
    //    - if db != nil && !cfg.DryRun {
    //         repo := processor.NewStagingRepository(db)
    //         for each successful result:
    //             metadata := processor.ReadAlbumMetadata(result.MetadataFile)
    //             repo.CreateStagingItemFromResult(result, metadata)
    //      }

    // 4) Return an error if the overall job failed (or partial failures as you see fit).
    //    - Log a structured summary with scan ID, inbound_path, staging_path, albums_total, albums_success, albums_failed, duration, dry_run, rate_limit.

    return nil
}
```

  > Note: inbound and staging paths are resolved from the configured libraries (type `inbound` and `staging`) and are not part of `StagingJobConfig`.

### DRY Considerations

- **Reuse, not re-implement**:
  - Use existing public APIs from `internal/scanner`:
    - `NewScanDB`, `OpenScanDB`, `ScanDB.ComputeAlbumGrouping`, `ScanDB.GetStats`, `NewFileScanner`, `FileScanner.ScanDirectory`.
  - Use existing public APIs from `internal/processor`:
    - `NewProcessor`, `Processor.ProcessAllAlbums`, `ReadAlbumMetadata`, `NewStagingRepository`, `StagingRepository.CreateStagingItemFromResult`, `GetProcessStats`.
- **Separate orchestration from CLIs**:
  - The orchestration above is the **single source of truth** for “one staging run” from the worker’s perspective.
  - Optionally, the CLI `main.go` files for `scan-inbound` and `process-scan` could later be refactored to call smaller helpers (e.g. `RunScanOnce`, `RunProcessOnce`) that share structure with `RunStagingJobCycle`, but that is not required for the cron job itself.

---

## Worker Integration

### 1. Config Wiring

- Extend `config.AppConfig` with a new struct, e.g. `StagingScanConfig`.
- Add it to `AppConfig` and `LoadConfig()`.
- Ensure the worker can read:
  - `cfg.StagingScan.Enabled`
  - `cfg.StagingScan.Schedule`
  - `cfg.StagingScan.Workers`, `RateLimit`, `DryRun`, and any scan output directory setting.

At runtime, the worker should read the latest values for `staging_scan.*` and library configuration at the beginning of each run so that changes made via the Admin UI take effect without a service restart.

### 2. Scheduling Strategy Options

#### Option A: Simple Interval (Ticker)

- In `worker.NewWorkerServer()` or right after creating the worker, start a goroutine if `cfg.StagingScan.Enabled` is `true`:

```go
if cfg.StagingScan.Enabled {
  d, err := time.ParseDuration(cfg.StagingScan.Interval)
    if err != nil {
        // log and disable cron, or fall back to default
    } else {
        go func() {
            ticker := time.NewTicker(d)
            defer ticker.Stop()

            for {
                select {
                case <-ticker.C:
                    // create context with timeout/cancel
                    ctx, cancel := context.WithTimeout(context.Background(), d)
                    err := workflow.RunStagingJobCycle(ctx, jobCfgFromAppConfig(cfg), dbManager.GetGormDB(), logger)
                    cancel()
                    // log success/failure

                case <-shutdownChan:
                    return
                }
            }
        }()
    }
}
```

- Pros:
  - Minimal dependencies.
  - Easy to reason about.
- Cons:
  - Fixed interval only; no cron expressions.

#### Option B: Cron Expressions (e.g. `robfig/cron`)

- Add a dependency on a cron library (if acceptable for you).
- Use `cfg.StagingScan.Schedule` (e.g. `"0 */1 * * *"`):

```go
c := cron.New()
_, err := c.AddFunc(cfg.StagingScan.Schedule, func() {
  ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
    defer cancel()
  // Guard against overlapping runs
  if !stagingJobRunning.CompareAndSwap(false, true) {
    logger.Warn("staging scan: previous run still in progress, skipping this tick")
    return
  }
  defer stagingJobRunning.Store(false)

  _ = workflow.RunStagingJobCycle(ctx, jobCfgFromAppConfig(cfg), dbManager.GetGormDB(), logger)
})
if err != nil {
    // log and skip cron
} else {
    c.Start()
}
```

- Pros:
  - Flexible cron syntax.
- Cons:
  - Extra dependency; slightly more moving parts.

#### Option C: Asynq-based Task Scheduling

- Define a new task type in `internal/media`, e.g. `TypeStagingScan`.
- Add handler in `worker/main.go` mux:

```go
mux.HandleFunc(media.TypeStagingScan, func(ctx context.Context, t *asynq.Task) error {
  return workflow.RunStagingJobCycle(ctx, jobCfgFromTask(t), dbManager.GetGormDB(), logger)
})
```

- Implement a simple scheduler (either:
  - inside worker using a ticker that enqueues `TypeStagingScan`, or
  - externally via another service / cron that calls the API or Redis directly).

- Pros:
  - Leverages existing Asynq infra.
  - Better observability (job queues, retries, dead-letter queues).
- Cons:
  - Slightly more complex wiring.

---

## Error Handling & Observability

- **Logging**:
  - Log at the start and end of each cron run, with key fields:
    - inbound path, staging root, scan DB path, duration, number of albums/items processed, and counts of successes/failures.
- **Partial failures**:
  - `RunStagingJobOnce` should:
    - Return a non-nil error for “hard” failures (e.g., inbound path missing, DB cannot be opened).
    - For album-level errors, log warnings but continue processing other albums.
- **Backoff / Overlap**:
  - Decide what should happen if a new tick occurs while a previous run is still in progress.
    - Simplest approach: guard with a mutex or atomic flag so only one run is active at a time.

---

## Open Decisions (for you to edit)

These are intentionally left as design levers for you to choose:

1. **Package & file naming**
   - `internal/workflow` vs `internal/media` vs `internal/processing`.
   - Single `staging_job.go` vs splitting into multiple smaller helpers.

2. **Scheduling backend**
   - `time.Ticker`-based interval vs cron expressions vs Asynq-based scheduling.

3. **Scan DB lifecycle**
   - Whether to keep or clean up scan DB files after processing.
   - Whether to store the generated scan DB path or ID somewhere for debugging.

4. **Rate limiting & workers**
   - How aggressive the defaults should be for `Workers` and `RateLimit`.
   - Whether to tie them to `cfg.Processing.ScanWorkers` or have independent values.

5. **Dry-run mode in cron**
   - Whether to support a `dry_run` flag in config for testing the cron job without moving files.

---

## Summary

- The cron-driven staging workflow should:
  - Run entirely in-process inside the worker.
  - Reuse existing `internal/scanner` and `internal/processor` packages instead of calling binaries.
  - Centralize the orchestration logic in a shared helper like `RunStagingJobCycle` to avoid DRY violations.
  - Be driven by a config-backed scheduler (ticker / cron / Asynq) that you can adjust without code changes.

This document is meant as a starting point; please adjust naming, config shape, and scheduling strategy to align with how you want Melodee’s worker to evolve.