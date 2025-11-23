# System Runbooks

This document contains operational runbooks for common scenarios in the Melodee system.

## Table of Contents
1. [Library Onboarding](#library-onboarding)
2. [DLQ Spike Handling](#dlq-spike-handling)
3. [Failed Scan Recovery](#failed-scan-recovery)
4. [Performance Issues](#performance-issues)
5. [Database Connection Problems](#database-connection-problems)
6. [Capacity Monitoring](#capacity-monitoring)

## Library Onboarding

### Objective
Add a new music library to the system and ensure it processes correctly.

### Steps
1. **Prepare the library**:
   - Ensure the music files are accessible at the designated path
   - Verify file permissions (should be readable by the melodee service)
   - Check for any unusual file names or paths that might cause issues

2. **Add the library via admin UI**:
   - Navigate to Admin Dashboard → Libraries
   - Click "Add Library"
   - Select the library type (inbound/staging/production)
   - Specify the path to the music files
   - Configure any special processing rules if needed

3. **Initiate the initial scan**:
   - Click "Scan Now" for the newly added library
   - Monitor the scan progress in the UI
   - Verify scan completes without errors

4. **Monitor processing pipeline**:
   - Check inbound files appear in quarantine section if using inbound library type
   - Verify processing moves files through staging to production as appropriate
   - Monitor capacity metrics during processing

5. **Validate content**:
   - Verify artists/albums appear in browsing interfaces
   - Test playback of sampled tracks
   - Check metadata accuracy

### Expected Outcomes
- New library appears in UI
- Files scanned and processed according to pipeline rules
- Content accessible via UI and API clients
- No errors in processing logs

### Troubleshooting
- If scan fails:
  - Check file permissions at the library path
  - Verify disk space availability
  - Review application logs for specific error messages
- If files stay in quarantine:
  - Check quarantine reason codes in admin UI
  - Address specific issues (checksum errors, tag problems, etc.)

### Rollback Plan
- Remove the library via admin UI
- Clean up any partially processed files manually if needed
- Restart the application if there are lingering resource locks

---

## DLQ Spike Handling

### Objective
Detect and resolve spikes in dead letter queue items.

### Symptoms
- Grafana dashboard shows sharp increase in DLQ size
- Processing jobs are failing at higher than normal rate
- Error logs contain processing failures

### Detection
1. **Monitor the dashboard**:
   - Check "Job Queues" panel for increasing DLQ sizes
   - Look for "Failed Jobs" panel showing spikes
   - Monitor error rate graphs

2. **Check logs**:
   - Look for repeated error patterns in application logs
   - Check for connection timeouts, out-of-memory errors, or processing failures

### Resolution Steps
1. **Immediate triage**:
   - Access the admin DLQ management UI
   - Identify the type of jobs failing (which queue)
   - Look for common patterns in failure reasons

2. **Assess severity**:
   - If DLQ size is growing rapidly (>100 items in 5 minutes), consider pausing non-critical operations
   - Check if the same job type or library is affected

3. **Fix root cause**:
   - For file processing errors: check file integrity, permissions, or format issues
   - For database errors: check connection pools, deadlocks, or constraint violations
   - For external service errors: check connectivity and service availability

4. **Clear DLQ**:
   - For fixed issues: requeue appropriate items
   - For irrecoverable errors: purge failed items
   - Monitor DLQ size to ensure it decreases

5. **Prevention**:
   - Adjust retry policies if appropriate
   - Implement circuit breakers if external services are involved
   - Update monitoring alerts for early detection

### Post-Incident Actions
- Document root cause and resolution
- Update monitoring if needed to catch similar issues earlier
- Consider adjusting retry/backoff strategies if appropriate

---

## Failed Scan Recovery

### Objective
Identify and resolve issues with library scanning operations.

### Symptoms
- Scans show "failed" status or stall
- Processing pipeline stalls
- Files not appearing as expected
- High error rate in logs during scanning

### Diagnosis
1. **Check scan status**:
   - Go to Admin Dashboard → Libraries
   - Look for libraries showing "scanning" for extended periods or "failed"
   - Note the last scan timestamp and status

2. **Review logs**:
   - Filter logs by timestamp of failed scan
   - Look for specific error messages like file permission errors, I/O errors, etc.
   - Check for patterns in failed file paths

3. **Check system resources**:
   - Monitor CPU, memory, and disk usage during scan
   - Check if scan is hitting any system limits

### Recovery Steps
1. **Identify the cause**:
   - Permission errors: fix file/directory permissions
   - Disk full: free up space
   - Corrupted files: quarantine or remove problematic files
   - Database lockups: restart if necessary

2. **Clean up**:
   - If scan partially completed, consider manual cleanup of inconsistent state
   - Remove any lock files if they exist

3. **Retry the scan**:
   - Use Admin UI to trigger a new scan
   - Consider scanning a subset if the full library is problematic

4. **Monitor recovery**:
   - Watch logs during the retry
   - Monitor processing metrics to ensure pipeline continues working

### Prevention
- Regular maintenance scans to catch permission issues early
- Monitor disk space before starting large scans
- Implement progressive or incremental scans for large libraries

---

## Performance Issues

### Objective
Diagnose and resolve performance degradation.

### Symptoms
- Slow API responses
- High memory/CPU usage
- Slow UI responses
- Playback buffering issues

### Investigation Steps
1. **Check metrics dashboard**:
   - Look for elevated response times
   - Check for resource saturation (CPU, memory, disk I/O)
   - Monitor database query times

2. **Review logs**:
   - Look for slow query warnings
   - Check for garbage collection warnings
   - Monitor for resource exhaustion errors

3. **Analyze specific components**:
   - Identify if issue is API, transcoding, or database related
   - Check concurrent stream limits

### Resolution
1. **Immediate fixes**:
   - Restart services if there are memory leaks
   - Scale up resources if temporarily constrained
   - Temporarily disable non-critical processing

2. **Long-term fixes**:
   - Optimize slow queries
   - Tune transcoding profiles
   - Adjust connection pool sizes
   - Implement better caching if needed

---

## Database Connection Problems

### Objective
Handle database connection issues and resource exhaustion.

### Symptoms
- API requests timing out
- "Connection refused" errors in logs
- High database connection counts
- Slow query responses

### Diagnostics
1. **Check connection metrics**:
   - Look at "Database Connections" panel in Grafana
   - Check for connections approaching limits

2. **Review database logs**:
   - Check for maximum connection limit reached
   - Look for long-running queries causing connection buildup

### Resolution
1. **Immediate actions**:
   - Check for long-running transactions
   - Kill any problematic queries if needed
   - Restart application if connections are leaked

2. **Configuration changes**:
   - Adjust connection pool sizes
   - Tune query timeouts
   - Consider read replicas for read-heavy operations

---

## Capacity Monitoring

### Objective
Monitor and respond to capacity constraints.

### Monitoring Points
1. **Disk Space**:
   - Check the "Capacity Usage" panel in Grafana
   - Monitor individual library capacity percentages
   - Watch for approaching limits

2. **Processing Capacity**:
   - Monitor transcoding queue lengths
   - Check for backlog builds in various processing queues

### Actions for Capacity Issues
1. **Near Limits (80%)**:
   - Prepare for scaling by identifying potential expansion options
   - Review data retention policies

2. **Critical (90%+)**:
   - Alert operators immediately
   - Consider disabling non-critical processing
   - Plan for immediate capacity expansion