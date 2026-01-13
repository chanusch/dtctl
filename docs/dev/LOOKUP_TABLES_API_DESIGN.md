# Lookup Tables API Design

## Overview

Lookup tables in Dynatrace are **tabular files** stored in the Grail Resource Store that can be loaded and joined with observability data in DQL queries for data enrichment. This document describes the dtctl API for managing these lookup tables.

### Key Use Cases
- **Data Enrichment**: Join lookup tables with logs, metrics, spans for context
- **Configuration Management**: Store and version reference data (IP ranges, service mappings, error codes)
- **Testing & CI/CD**: Programmatically manage test data sets
- **Bulk Operations**: Upload/download multiple lookup tables

---

## Design Principles Alignment

Following dtctl's core design principles:

1. **✅ Verb-Noun Pattern**: `dtctl <verb> lookup [options]`
2. **✅ kubectl-like UX**: Familiar commands (get, describe, create, delete, apply)
3. **✅ YAML Input, Multiple Outputs**: Accept CSV/YAML, output as table/JSON/YAML/CSV
4. **✅ No Leaky Abstractions**: Direct mapping to Grail Resource Store API
5. **✅ AI-Native**: Support `--plain` flag for machine parsing
6. **✅ Resource-Oriented**: Treat lookup tables as first-class resources

---

## Resource Definition

### Resource Name
- **Primary**: `lookup` / `lookups`
- **Short aliases**: `lkup`, `lu`
- **Type identifier**: Tabular files in Grail Resource Store under `/lookups/` path

### File Path Constraints
Per the Grail Resource Store API specification:
- Must start with `/lookups/`
- Only alphanumeric `[a-zA-Z0-9]`, `-`, `_`, `.`, `/`
- Must end with `[a-zA-Z0-9]`
- Must contain at least two `/` characters
- Between any two `/` there must be at least one alphanumeric character
- Maximum 500 characters
- Example: `/lookups/grail/pm/error_codes.csv`

---

## Command Structure

### 1. List Lookups

List all lookup tables with metadata only (no data preview).

```bash
# List all lookup files
dtctl get lookups

# List with wide output (shows more metadata)
dtctl get lookups -o wide

# Output formats
dtctl get lookups -o json
dtctl get lookups -o yaml
dtctl get lookups -o csv
```

**Output (table format)**:
```
PATH                              DISPLAY NAME       SIZE      RECORDS   MODIFIED
/lookups/grail/pm/error_codes     Error Codes        24.5 KB   120       2h ago
/lookups/grail/pm/service_map     Service Mapping    156 KB    450       1d ago
/lookups/prod/ip_ranges           IP Ranges          8.2 KB    34        5d ago
```

**Implementation Notes**:
- Uses `fetch dt.system.files` DQL query to list files
- Filters for paths starting with `/lookups/`
- Extracts metadata: path, size, record count, modification time
- Similar pattern to existing `bucket` resource

### 2. Get Lookup

Retrieve a specific lookup table with data preview.

```bash
# Get lookup metadata with preview (first 10 rows)
dtctl get lookup /lookups/grail/pm/error_codes

# Download lookup data as CSV
dtctl get lookup /lookups/grail/pm/error_codes -o csv > error_codes.csv

# Download as JSON records
dtctl get lookup /lookups/grail/pm/error_codes -o json
```

**Implementation Notes**:
- Uses `load "<path>"` DQL query to fetch data
- Shows first 10 rows by default
- Full data export available via `-o csv` or `-o json`

### 3. Describe Lookup

Show detailed metadata without full data.

```bash
# Get lookup metadata
dtctl describe lookup /lookups/grail/pm/error_codes

# JSON output
dtctl describe lookup /lookups/grail/pm/error_codes -o json
```

**Describe Output**:
```
Path:         /lookups/grail/pm/error_codes
Display Name: Error Codes
Description:  HTTP error code descriptions
File Size:    24.5 KB
Records:      120
Lookup Field: code
Columns:      code, message, severity
Created:      2024-01-10 14:23:45 UTC
Modified:     2024-01-13 16:45:12 UTC

Data Preview (first 5 rows):
CODE  MESSAGE                  SEVERITY
200   OK                       info
400   Bad Request              error
401   Unauthorized             error
403   Forbidden                error
404   Not Found                warning
```

### 4. Create Lookup

Upload a new lookup table. Fails if the table already exists (use `apply` for idempotent operations).

```bash
# Create from CSV file (auto-detect headers)
dtctl create lookup -f error_codes.csv \
  --path /lookups/grail/pm/error_codes \
  --lookup-field code \
  --display-name "Error Codes" \
  --description "HTTP error code descriptions"

# Create from manifest (YAML)
dtctl create lookup -f lookup-manifest.yaml

# Create with custom parse pattern (override auto-detection)
dtctl create lookup -f data.txt \
  --path /lookups/custom/data \
  --lookup-field id \
  --parse-pattern "LD:id '|' LD:value '|' LD:timestamp" \
  --skip-records 1

# Prevent overwrite (default behavior)
dtctl create lookup -f data.csv --path /lookups/test
```

**Manifest Example** (`lookup-manifest.yaml`):
```yaml
apiVersion: grail/v1
kind: Lookup
metadata:
  path: /lookups/grail/pm/error_codes
  displayName: Error Codes
  description: HTTP error code descriptions
spec:
  lookupField: code
  parsePattern: "LD:code ',' LD:message ',' LD:severity"  # Auto-detected for CSV
  skippedRecords: 1  # Skip header row
  autoFlatten: true
  timezone: UTC
  locale: en_US
  overwrite: false
data:
  source: error_codes.csv  # Path to data file
```

**CSV Auto-Detection**:
- Reads first row as column headers
- Generates DPL parse pattern: `LD:col1 ',' LD:col2 ',' LD:col3`
- Sets `skippedRecords: 1` to skip header row
- Can be overridden with `--parse-pattern` flag

**Implementation Notes**:
- Multipart form upload to `/platform/storage/resource-store/v1/files/tabular/lookup:upload`
- Auto-detect CSV format and generate parse pattern
- Support custom DPL parse patterns for non-CSV formats
- Validate file path constraints
- Default `overwrite: false` to prevent accidental data loss

### 5. Apply Lookup

Create or update lookup table (idempotent operation).

```bash
# Apply from manifest (create or update)
dtctl apply -f lookup-manifest.yaml

# Update from CSV (overwrite if exists)
dtctl apply -f error_codes.csv --path /lookups/grail/pm/error_codes

# Apply with template variables
dtctl apply -f lookup-template.yaml --set environment=prod

# Dry-run to preview changes
dtctl apply -f lookup-manifest.yaml --dry-run
```

**Implementation Notes**:
- Check if lookup exists using DQL query
- If exists: set `overwrite: true` and update
- If not exists: create new
- Idempotent operation (same as other dtctl resources)

### 6. Edit Lookup

Edit lookup data interactively in `$EDITOR`.

```bash
# Edit lookup in default format (CSV)
dtctl edit lookup /lookups/grail/pm/error_codes

# Edit in specific format
dtctl edit lookup /lookups/grail/pm/error_codes --format csv
dtctl edit lookup /lookups/grail/pm/error_codes --format json
```

**Workflow**:
1. Download current data as CSV (default) or JSON
2. Open in `$EDITOR` (vim, nano, etc.)
3. On save: validate, show diff, confirm upload
4. Upload updated data with `overwrite: true`

**Implementation Notes**:
- Similar to `dtctl edit workflow` pattern
- Default format: CSV (human-friendly for tabular data)
- Validate on save before uploading

### 7. Delete Lookup

Delete a lookup table (irreversible operation).

```bash
# Delete by path (requires confirmation)
dtctl delete lookup /lookups/grail/pm/error_codes

# Delete with confirmation skip
dtctl delete lookup /lookups/grail/pm/error_codes -y

# Delete from manifest
dtctl delete -f lookup-manifest.yaml

# Dry-run
dtctl delete lookup /lookups/test --dry-run
```

**Output**:
```
Are you sure you want to delete the following lookup table?
  Path:         /lookups/grail/pm/error_codes
  Display Name: Error Codes
  Records:      120
  Size:         24.5 KB

Type 'yes' to confirm: yes
Lookup table deleted successfully.
```

**Implementation Notes**:
- POST to `/platform/storage/resource-store/v1/files:delete`
- Requires confirmation by default (use `-y` to skip)
- Irreversible operation - warn user clearly
- Support `--plain` mode for scripting

---

## Design Decisions

Based on user feedback and dtctl patterns:

### 1. CSV Auto-detection with Override
- **Decision**: Auto-detect CSV headers and generate parse patterns by default
- **Override**: Allow `--parse-pattern` flag to specify custom patterns
- **Rationale**: Makes common CSV use cases trivial while supporting advanced scenarios

### 2. CSV Edit Format
- **Decision**: Default to CSV format for `dtctl edit lookup`
- **Rationale**: CSV is human-friendly and works well in text editors for tabular data
- **Alternative**: JSON/YAML available via `--format` flag

### 3. Metadata-only List
- **Decision**: `dtctl get lookups` shows only metadata (no data preview)
- **Rationale**: Fast, clean output. Use `describe` or individual `get` for data
- **Performance**: Avoids loading data for every lookup table

### 4. Full Paths Only
- **Decision**: Require complete paths like `/lookups/grail/pm/error_codes`
- **Rationale**: Explicit, no ambiguity, matches API specification
- **Alternative**: No shortcuts or relative paths to avoid confusion

### 5. No Post-upload Validation
- **Decision**: Trust API validation, no additional queries after upload
- **Rationale**: Fast, simple. API validates during upload
- **Alternative**: Could add `--validate` flag for optional verification

---

## API Endpoints

### Grail Resource Store API

**Base URL**: `/platform/storage/resource-store/v1`

**Endpoints**:
- `POST /files/tabular/lookup:upload` - Upload lookup data
- `POST /files/tabular/lookup:test-pattern` - Test parse pattern
- `POST /files:delete` - Delete file

**DQL Queries** (via Grail Query API):
- `fetch dt.system.files | filter path starts_with "/lookups/"` - List lookups
- `load "<path>"` - Load lookup data

### Required Scopes

- **Read operations**: `storage:files:read`
- **Write operations** (create/update): `storage:files:write`
- **Delete operations**: `storage:files:delete`

---

## Example Workflows

### 1. Create and Query a Lookup Table

```bash
# Prepare CSV file (error_codes.csv)
# code,message,severity
# 200,OK,info
# 400,Bad Request,error
# 404,Not Found,warning
# 500,Internal Server Error,critical

# Upload lookup table
dtctl create lookup -f error_codes.csv \
  --path /lookups/grail/pm/error_codes \
  --lookup-field code \
  --display-name "Error Codes"

# Verify upload
dtctl get lookup /lookups/grail/pm/error_codes

# Use in DQL query
dtctl query "
  fetch logs
  | lookup [load '/lookups/grail/pm/error_codes'], lookupField:status_code
  | fields timestamp, status_code, message, severity
  | filter severity == 'critical'
"
```

### 2. Edit Existing Lookup

```bash
# Edit lookup in terminal
dtctl edit lookup /lookups/grail/pm/error_codes

# (Opens CSV in $EDITOR, user makes changes, saves)

# Changes are automatically uploaded on save
```

### 3. Backup and Restore

```bash
# Backup single lookup
dtctl get lookup /lookups/grail/pm/error_codes -o yaml > backup.yaml

# Backup all lookups metadata
dtctl get lookups -o json > all-lookups.json

# Restore from backup
dtctl apply -f backup.yaml
```

### 4. CI/CD Integration

```bash
#!/bin/bash
# deploy-lookups.sh

# Validate all lookups before deploying
for file in lookups/*.csv; do
  manifest="${file%.csv}.yaml"
  if [ -f "$manifest" ]; then
    dtctl apply -f "$manifest" --dry-run || exit 1
  fi
done

# Deploy lookups
for manifest in lookups/*.yaml; do
  dtctl apply -f "$manifest" --plain || exit 1
done

# Verify deployment
dtctl get lookups --plain -o json | \
  jq '.[] | select(.path | startswith("/lookups/prod/"))'
```

### 5. Custom Parse Patterns (Non-CSV)

```bash
# Pipe-delimited file
dtctl create lookup -f data.txt \
  --path /lookups/custom/pipe_delimited \
  --lookup-field id \
  --parse-pattern "LD:id '|' LD:name '|' LD:value"

# Fixed-width format
dtctl create lookup -f fixed_width.txt \
  --path /lookups/custom/fixed \
  --lookup-field id \
  --parse-pattern "SPACE? LD:id:length=10 SPACE? LD:name:length=20 SPACE? LD:value"
```

---

## Implementation Plan

### Phase 1: Core CRUD Operations
1. ✅ Create `pkg/resources/lookup/` package
2. ✅ Implement `lookup.go` handler with List, Get, Create, Delete
3. ✅ Implement `csv.go` for CSV parsing and pattern generation
4. ✅ Implement `multipart.go` for multipart form uploads
5. ✅ Add commands to `cmd/get.go`, `cmd/describe.go`, `cmd/create.go`, `cmd/delete.go`

### Phase 2: Advanced Features
6. ✅ Implement `dtctl apply` for idempotent operations
7. ✅ Implement `dtctl edit` with CSV format support
8. ✅ Add output formatters (table, JSON, YAML, CSV)
9. ✅ Add shell completion support

### Phase 3: Testing & Documentation
10. ✅ Add unit tests (`lookup_test.go`, `csv_test.go`)
11. ✅ Add integration tests
12. ✅ Update API_DESIGN.md
13. ✅ Update IMPLEMENTATION_STATUS.md
14. ✅ Update README.md with examples

---

## Resource Handler Structure

Following dtctl patterns:

```go
package lookup

type Handler struct {
    client *client.Client
}

// Lookup represents a lookup table file
type Lookup struct {
    Path         string   `json:"path" table:"PATH"`
    DisplayName  string   `json:"displayName,omitempty" table:"DISPLAY_NAME"`
    Description  string   `json:"description,omitempty" table:"DESCRIPTION,wide"`
    FileSize     int64    `json:"fileSize,omitempty" table:"SIZE"`
    Records      int      `json:"records,omitempty" table:"RECORDS"`
    LookupField  string   `json:"lookupField" table:"LOOKUP_FIELD,wide"`
    Columns      []string `json:"columns,omitempty" table:"-"`
    Modified     string   `json:"modified,omitempty" table:"MODIFIED"`
}

// Core operations
func (h *Handler) List() ([]Lookup, error)
func (h *Handler) Get(path string) (*Lookup, error)
func (h *Handler) GetData(path string) ([]map[string]interface{}, error)
func (h *Handler) Create(req CreateRequest) error
func (h *Handler) Update(path string, req UpdateRequest) error
func (h *Handler) Delete(path string) error
```

---

## Error Handling

Following dtctl patterns:

```go
// Specific error types
var (
    ErrLookupNotFound      = errors.New("lookup table not found")
    ErrLookupExists        = errors.New("lookup table already exists")
    ErrInvalidPath         = errors.New("invalid file path")
    ErrInvalidParsePattern = errors.New("invalid parse pattern")
    ErrFileTooLarge        = errors.New("file size exceeds maximum limit")
)

// User-friendly error messages
if resp.StatusCode == 404 {
    return fmt.Errorf("lookup table %q not found. Run 'dtctl get lookups' to list available lookups", path)
}

if resp.StatusCode == 409 {
    return fmt.Errorf("lookup table %q already exists. Use 'dtctl apply' to update or add --overwrite flag", path)
}
```

---

## Benefits

1. **✅ Consistency**: Follows dtctl's established patterns (verb-noun, output formats)
2. **✅ Simplicity**: Auto-detection reduces complexity for common CSV use cases
3. **✅ Flexibility**: Custom parse patterns support non-CSV formats
4. **✅ Speed**: No post-upload validation keeps operations fast
5. **✅ Safety**: Confirmation prompts prevent accidental deletions
6. **✅ Integration**: Works seamlessly with existing DQL query workflows
7. **✅ Automation**: Machine-readable output enables scripting

---

## Future Enhancements

Potential features for future consideration:

1. **Bulk Operations**: `dtctl apply -f lookups/ --recursive` for directory uploads
2. **Version History**: Support versioning/snapshots like documents
3. **Diff Visualization**: Side-by-side diffs for data changes
4. **Watch Mode**: `dtctl get lookups --watch` for live updates
5. **Compression**: Support gzip-compressed uploads
6. **Test Command**: `dtctl test lookup` to validate parse patterns
7. **Validate Command**: `dtctl validate lookup` to check manifests

---

## References

- **API Specification**: `.api-spec/grail-resource-store.yaml`
- **DPL Documentation**: https://docs.dynatrace.com/docs/discover-dynatrace/references/dynatrace-pattern-language
- **Grail Documentation**: https://docs.dynatrace.com/docs/discover-dynatrace/platform/grail
- **Similar Resources**: `pkg/resources/bucket/` (storage management pattern)
