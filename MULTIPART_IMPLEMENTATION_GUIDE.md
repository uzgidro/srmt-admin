# Multipart/Form-Data Implementation Guide

## Overview

The incident handlers now support **both** `application/json` and `multipart/form-data` content types. This allows:
- Creating/editing incidents with file uploads in a single request
- Backward compatibility with existing JSON-only clients
- Transactional safety with automatic cleanup on failure

## Implementation Status

✅ **Completed for Incidents**
- `POST /api/incidents` - Add incident with files
- `PATCH /api/incidents/{id}` - Edit incident with files

🔜 **Pending** (same pattern can be applied):
- Discharges
- Shutdowns
- Visits

---

## How to Use

### 1. Create Incident with JSON (Existing Behavior)

```http
POST /api/incidents
Content-Type: application/json
Authorization: Bearer <token>

{
  "organization_id": 123,
  "incident_time": "2025-01-27T10:30:00Z",
  "description": "Water leak detected",
  "file_ids": [45, 46]  // Reference existing files
}
```

**Response:**
```json
{
  "status": "success",
  "id": 789
}
```

---

### 2. Create Incident with Multipart (NEW!)

```http
POST /api/incidents
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary
Authorization: Bearer <token>

------WebKitFormBoundary
Content-Disposition: form-data; name="organization_id"

123
------WebKitFormBoundary
Content-Disposition: form-data; name="incident_time"

2025-01-27T10:30:00Z
------WebKitFormBoundary
Content-Disposition: form-data; name="description"

Water leak detected in sector A
------WebKitFormBoundary
Content-Disposition: form-data; name="files"; filename="photo.jpg"
Content-Type: image/jpeg

<binary file data>
------WebKitFormBoundary
Content-Disposition: form-data; name="files"; filename="report.pdf"
Content-Type: application/pdf

<binary file data>
------WebKitFormBoundary--
```

**Response:**
```json
{
  "status": "success",
  "id": 789,
  "uploaded_files": [
    {
      "id": 100,
      "file_name": "photo.jpg",
      "object_key": "incident/2025/01/27/uuid-123.jpg",
      "size_bytes": 524288,
      "mime_type": "image/jpeg"
    },
    {
      "id": 101,
      "file_name": "report.pdf",
      "object_key": "incident/2025/01/27/uuid-456.pdf",
      "size_bytes": 1048576,
      "mime_type": "application/pdf"
    }
  ]
}
```

---

### 3. Mix Uploaded Files + Existing Files

```http
POST /api/incidents
Content-Type: multipart/form-data

organization_id=123
incident_time=2025-01-27T10:30:00Z
description=Water leak
file_ids=45,46          // Keep existing files
files=<new_photo.jpg>   // Upload new file
```

**Result:** Incident will have files `[45, 46, 100]` (existing + new)

---

### 4. Edit Incident - Replace ALL Files (JSON)

```http
PATCH /api/incidents/789
Content-Type: application/json

{
  "description": "Updated description",
  "file_ids": [50, 51, 52]  // Replace with these files
}
```

**Behavior:**
- Unlinks old files (45, 46)
- Links new files (50, 51, 52)

---

### 5. Edit Incident - Remove ALL Files (FIXED!)

```http
PATCH /api/incidents/789
Content-Type: application/json

{
  "description": "Updated description",
  "file_ids": []  // Empty array now works!
}
```

**Behavior:**
- Unlinks all old files
- Incident has no files

**Note:** Previously this didn't work because `len(file_ids) > 0` check prevented empty arrays.

---

### 6. Edit Incident - Don't Touch Files

```http
PATCH /api/incidents/789
Content-Type: application/json

{
  "description": "Updated description"
  // No file_ids field
}
```

**Behavior:**
- Updates description only
- Files remain unchanged

---

### 7. Edit Incident with Multipart + Upload

```http
PATCH /api/incidents/789
Content-Type: multipart/form-data

description=Updated description
file_ids=45         // Keep file 45
files=<new.jpg>     // Upload new file
```

**Behavior:**
- Unlinks ALL old files
- Uploads new file (gets ID 102)
- Links files: [45, 102]

---

## File Upload Details

### Storage Structure

Files are stored in MinIO with this path structure:
```
{category}/{year}/{month}/{day}/{uuid}.{ext}

Examples:
incident/2025/01/27/a1b2c3d4-uuid.jpg
incident/2025/01/27/e5f6g7h8-uuid.pdf
```

### Category Mapping

| Entity    | Category Name | Example Path                          |
|-----------|---------------|---------------------------------------|
| Incident  | `incident`    | `incident/2025/01/27/uuid.jpg`        |
| Discharge | `discharge`   | `discharge/2025/01/27/uuid.pdf`       |
| Shutdown  | `shutdown`    | `shutdown/2025/01/27/uuid.jpg`        |
| Visit     | `visit`       | `visit/2025/01/27/uuid.pdf`           |

### Date Selection

- **Add:** Uses entity's date (e.g., `incident_time` for incidents)
- **Edit:** Uses current timestamp (`time.Now()`)

### File Size Limit

Maximum upload size: **50 MB**

---

## Error Handling & Compensation

### Automatic Cleanup

If entity creation/update fails after files are uploaded, the system automatically:

1. **Deletes files from MinIO**
2. **Deletes file metadata from database**
3. **Returns error to client**

Example flow:
```
1. Upload file1.jpg → MinIO ✅ → DB ID=100 ✅
2. Upload file2.pdf → MinIO ✅ → DB ID=101 ✅
3. Create incident → FAILS ❌
4. Compensation kicks in:
   - Delete file1.jpg from MinIO ✅
   - Delete file2.pdf from MinIO ✅
   - Delete ID=100 from DB ✅
   - Delete ID=101 from DB ✅
5. Return error to client
```

This ensures **no orphaned files** in storage or database.

### Validation Failure

If validation fails (e.g., missing required field):
- Uploaded files are cleaned up
- Error response returned

### Partial Failures

If file linking fails after entity is created:
- Entity creation **succeeds**
- File linking **fails** (logged as error)
- Response indicates success with warning in logs

---

## Code Architecture

### Helper Packages

#### 1. `internal/lib/service/fileupload/helper.go`

**Functions:**
- `ProcessFormFiles()` - Uploads files from multipart form
- `CompensateEntityUpload()` - Cleanup on failure
- `uploadSingleFile()` - Upload individual file
- `compensateUploads()` - Rollback mechanism

**Interfaces:**
```go
type FileUploader interface {
    UploadFile(ctx, objectKey string, reader io.Reader, size int64, contentType string) error
    DeleteFile(ctx, objectKey string) error
}

type FileMetaSaver interface {
    AddFile(ctx, fileData filemodel.Model) (int64, error)
    DeleteFile(ctx, id int64) error
}
```

#### 2. `internal/lib/api/formparser/parser.go`

**Functions:**
- `GetFormInt64()` / `GetFormInt64Required()`
- `GetFormString()` / `GetFormStringRequired()`
- `GetFormTime()` / `GetFormTimeRequired()`
- `GetFormFileIDs()` - Parse comma-separated IDs
- `IsMultipartForm()` / `IsJSONRequest()`
- `HasFormField()` - Check if field exists

### Handler Pattern

```go
func Add(log, adder, uploader, saver) http.HandlerFunc {
    return func(w, r) {
        var req Request
        var fileIDs []int64
        var uploadResult *UploadResult

        // Detect content type
        if IsMultipartForm(r) {
            // Parse multipart
            req, uploadResult, err = parseMultipartRequest(...)
            existingIDs := GetFormFileIDs(r, "file_ids")
            fileIDs = append(existingIDs, uploadResult.FileIDs...)
        } else {
            // Parse JSON
            DecodeJSON(r.Body, &req)
            fileIDs = req.FileIDs
        }

        // Validate
        if err := validator.Struct(req); err != nil {
            if uploadResult != nil {
                CompensateEntityUpload(...) // Cleanup!
            }
            return Error(...)
        }

        // Create entity
        id, err := adder.AddEntity(...)
        if err != nil {
            if uploadResult != nil {
                CompensateEntityUpload(...) // Cleanup!
            }
            return Error(...)
        }

        // Link files
        if len(fileIDs) > 0 {
            adder.LinkFiles(ctx, id, fileIDs)
        }

        // Return success
        return Success(id, uploadResult.UploadedFiles)
    }
}
```

---

## Testing

### Test Cases Covered

1. ✅ JSON with existing file IDs
2. ✅ JSON without files
3. ✅ Multipart with uploaded files
4. ✅ Multipart with mixed (uploaded + existing)
5. ✅ Edit with JSON (replace files)
6. ✅ Edit with JSON (remove all files)
7. ✅ Edit with JSON (don't touch files)
8. ✅ Edit with multipart (upload + keep some)
9. ✅ Validation failure (cleanup triggered)
10. ✅ Entity creation failure (cleanup triggered)

---

## Frontend Integration

### JavaScript Example

```javascript
// Create incident with file upload
const formData = new FormData();
formData.append('organization_id', '123');
formData.append('incident_time', '2025-01-27T10:30:00Z');
formData.append('description', 'Water leak');

// Add multiple files
formData.append('files', fileInput.files[0]);
formData.append('files', fileInput.files[1]);

// Add existing file IDs
formData.append('file_ids', '45,46');

const response = await fetch('/api/incidents', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`
  },
  body: formData
});

const result = await response.json();
console.log('Created incident:', result.id);
console.log('Uploaded files:', result.uploaded_files);
```

---

## Migration Guide for Other Entities

To add multipart support to discharges/shutdowns/visits:

### Step 1: Update Handler Signature

```go
// Before
func Add(log *slog.Logger, adder EntityAdder) http.HandlerFunc

// After
func Add(log *slog.Logger, adder EntityAdder, uploader FileUploader, saver FileMetaSaver) http.HandlerFunc
```

### Step 2: Add Content-Type Detection

```go
if formparser.IsMultipartForm(r) {
    req, uploadResult, err = parseMultipartAddRequest(r, log, uploader, saver)
    // ...
} else {
    render.DecodeJSON(r.Body, &req)
    // ...
}
```

### Step 3: Implement Parse Function

```go
func parseMultipartAddRequest(...) (Request, *UploadResult, error) {
    // Parse form fields
    orgID, _ := formparser.GetFormInt64(r, "organization_id")
    date, _ := formparser.GetFormTimeRequired(r, "date", time.RFC3339)

    // Upload files
    uploadResult, _ := fileupload.ProcessFormFiles(
        ctx, r, log, uploader, saver,
        "entity_category", // discharge/shutdown/visit
        date,
    )

    return Request{...}, uploadResult, nil
}
```

### Step 4: Add Compensation

```go
// After validation failure
if uploadResult != nil {
    fileupload.CompensateEntityUpload(ctx, log, uploader, saver, uploadResult)
}

// After entity creation failure
if uploadResult != nil {
    fileupload.CompensateEntityUpload(ctx, log, uploader, saver, uploadResult)
}
```

### Step 5: Update Router

```go
r.Post("/entities", handler.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo))
```

### Step 6: Update Edit Handler

```go
// Fix "can't remove all files" issue
if req.FileIDs != nil {  // Check if field exists, not if length > 0
    shouldUpdateFiles = true
    fileIDs = req.FileIDs
}
```

---

## Benefits

1. ✅ **Single Request**: Upload + create in one call
2. ✅ **Transactional Safety**: Automatic cleanup on failure
3. ✅ **Backward Compatible**: JSON still works
4. ✅ **Flexible**: Mix uploaded + existing files
5. ✅ **Bug Fix**: Can now remove all files in edit
6. ✅ **Better UX**: Simpler frontend code
7. ✅ **Cleaner API**: More RESTful design

---

## Next Steps

1. Apply same pattern to **discharges**
2. Apply same pattern to **shutdowns**
3. Apply same pattern to **visits**
4. Add integration tests
5. Update API documentation
6. Update frontend to use multipart

---

## Questions?

See the design document: `DESIGN_MULTIPART_UPLOAD.md`

For implementation examples:
- `internal/http-server/handlers/incidents-handler/add.go`
- `internal/http-server/handlers/incidents-handler/edit.go`
