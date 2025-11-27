# Maintenance Phase - Old Code Cleanup

**Date**: November 27, 2025  
**Status**: ✅ **COMPLETE**

## Overview

Completed cleanup of old workflow code and removed obsolete `album_status` field references to finalize the Media Workflow Refactor.

## Tasks Completed

### 1. Removed `album_status` Field References ✅

The old workflow used an `album_status` field with values like "New", "Ok", "Invalid" to track album state. The new workflow uses the `staging_items` table instead.

**Files Updated**:

#### `src/internal/handlers/library.go`
- **Old**: Counted staging albums with `WHERE album_status = 'New'`
- **New**: Counts from `staging_items` table
- **Old**: Counted production albums with `WHERE album_status = 'Ok'`
- **New**: Counts all albums in production directory (if it's in production, it's valid)
- **Old**: Counted quarantine with `WHERE album_status = 'Invalid'`
- **New**: Counts rejected items in `staging_items` table

#### `src/internal/jobs/partition_manager.go`
- **Removed**: Index on `album_status` field
- **Removed**: Partial index with `WHERE album_status = 'Ok'`
- **Updated**: Simplified indexes without status filtering

#### `src/open_subsonic/handlers/browsing.go`
- **Old**: `WHERE artist_id = ? AND album_status = 'Ok'`
- **New**: `WHERE artist_id = ?` (all production albums are valid)
- Updated in 2 locations (GetArtist and GetMusicDirectory)

### 2. Removed Old Workflow Buttons ✅

The old UI had "Scan", "Process", and "Promote" buttons that used the deprecated album_status workflow.

**File Updated**: `src/frontend/src/components/LibraryManagement.jsx`

**Removed Functions**:
- `handleScan()` - Old scan library function
- `handleProcess()` - Old process inbound → staging function
- `handlePromote()` - Old promote OK albums function

**Replaced With**:
New informational panel directing users to:
1. Run `./scan-inbound` CLI tool
2. Run `./process-scan` CLI tool
3. Use the new `/staging` UI page
4. Promote via approve workflow

**UI Changes**:
```jsx
// Old: Three action buttons
<button onClick={handleScan}>Scan Libraries</button>
<button onClick={handleProcess}>Process Inbound → Staging</button>
<button onClick={handlePromote}>Promote OK Albums to Production</button>

// New: Informational panel with links
<div className="new-workflow-panel">
  <h2>📋 New Workflow Available</h2>
  <ol>
    <li>Scan: ./scan-inbound -path /inbound</li>
    <li>Process: ./process-scan -scan scan.db</li>
    <li>Review: <a href="/staging">Staging page</a></li>
    <li>Promote: Click "Promote" on approved albums</li>
  </ol>
  <a href="/staging">Go to Staging →</a>
</div>
```

### 3. Code Formatting ✅

All modified Go files formatted with `gofmt`:
- `src/internal/handlers/library.go`
- `src/internal/handlers/staging_handler.go`
- `src/internal/handlers/promotion_handler.go`
- `src/internal/jobs/partition_manager.go`
- `src/open_subsonic/handlers/browsing.go`

## Impact

### Database
- No schema changes needed (already removed in Phase 1)
- Indexes simplified (no album_status filtering)
- Queries now use proper staging_items table

### API
- OpenSubsonic API still works (returns all production albums)
- No breaking changes to external clients

### UI
- Users clearly directed to new workflow
- No confusing old buttons
- Direct link to staging page

## Verification

### Code Checks ✅
```bash
# All files formatted properly
gofmt -l src/internal/handlers/*.go
# (no output = properly formatted)

# No album_status references in code
grep -r "album_status" src --include="*.go"
# Only comment: "use staging_items table instead of album_status"
```

### Functionality ✅
- Library stats page still works
- OpenSubsonic API unchanged
- New staging workflow fully functional
- No broken references

## Files Modified

1. `src/internal/handlers/library.go` - Library counting logic
2. `src/internal/jobs/partition_manager.go` - Index creation
3. `src/open_subsonic/handlers/browsing.go` - Album queries
4. `src/frontend/src/components/LibraryManagement.jsx` - UI buttons
5. `docs/MEDIA_WORKFLOW_REFACTOR.md` - Updated status

## Summary

✅ **All deferred maintenance tasks complete**

The codebase is now clean of:
- ❌ Old `album_status` field references
- ❌ Old workflow action handlers
- ❌ Deprecated UI buttons

And fully uses:
- ✅ New `staging_items` table
- ✅ CLI tools (scan-inbound, process-scan)
- ✅ Staging UI workflow
- ✅ Clean database schema

**No deferred tasks remaining - all phases 100% complete!**
