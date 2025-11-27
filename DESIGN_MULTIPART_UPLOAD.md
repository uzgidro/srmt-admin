# Design: Multipart/Form-Data Support for Entity Add/Edit

## Current Architecture

### File Upload Flow
1. **Standalone upload** at `/api/files/upload` (POST multipart/form-data)
   - Uploads file to MinIO
   - Saves metadata to `files` table
   - Returns file ID
2. **Entity creation** with `file_ids: [1, 2, 3]` (POST JSON)
   - Creates entity
   - Links existing file IDs via junction tables

### Current Limitations
- **Two-step process**: Upload files first, then create entity
- **No transactional safety**: Files can be uploaded but entity creation fails
- **Orphaned files**: If entity creation fails, files remain in storage
- **Poor UX**: Frontend must handle two separate API calls

## Proposed Solution

### Support Both Content Types

Handlers should accept:
1. **`application/json`** (current behavior)
   - Send `file_ids: [1, 2, 3]` for existing files
2. **`multipart/form-data`** (new feature)
   - Upload files + entity data in single request
   - Automatically create file records and link them

### Request Examples

#### Option 1: JSON (existing files)
```http
POST /api/incidents
Content-Type: application/json

{
  "organization_id": 123,
  "incident_time": "2025-01-15T10:30:00Z",
  "description": "Water leak",
  "file_ids": [45, 46, 47]
}
```

#### Option 2: Multipart (upload new files)
```http
POST /api/incidents
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary

------WebKitFormBoundary
Content-Disposition: form-data; name="organization_id"

123
------WebKitFormBoundary
Content-Disposition: form-data; name="incident_time"

2025-01-15T10:30:00Z
------WebKitFormBoundary
Content-Disposition: form-data; name="description"

Water leak detected
------WebKitFormBoundary
Content-Disposition: form-data; name="files"; filename="photo1.jpg"
Content-Type: image/jpeg

<binary data>
------WebKitFormBoundary
Content-Disposition: form-data; name="files"; filename="report.pdf"
Content-Type: application/pdf

<binary data>
------WebKitFormBoundary--
```

#### Option 3: Multipart (mix existing + new files)
```http
POST /api/incidents
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary

------WebKitFormBoundary
Content-Disposition: form-data; name="organization_id"

123
------WebKitFormBoundary
Content-Disposition: form-data; name="file_ids"

45,46
------WebKitFormBoundary
Content-Disposition: form-data; name="files"; filename="new_photo.jpg"
Content-Type: image/jpeg

<binary data>
------WebKitFormBoundary--
```

## Implementation Strategy

### 1. Create File Upload Helper

```go
// internal/lib/service/fileupload/helper.go
package fileupload

import (
	"context"
	"net/http"
)

type UploadedFile struct {
	ID        int64
	FileName  string
	ObjectKey string
}

// ProcessFormFiles handles file uploads from multipart form
// - Uploads each file to MinIO
// - Saves metadata to DB
// - Returns file IDs
// - Implements compensation on failure (deletes uploaded files)
func ProcessFormFiles(
	ctx context.Context,
	r *http.Request,
	uploader FileUploader,
	saver FileMetaSaver,
	categoryID int64,
	uploadDate time.Time,
) ([]int64, error)
```

### 2. Parse Form Data Helper

```go
// internal/lib/api/formparser/parser.go
package formparser

// ParseEntityForm extracts entity fields from multipart form
func ParseEntityForm(r *http.Request) (map[string]interface{}, error)

// GetFormInt64 safely parses int64 from form
func GetFormInt64(r *http.Request, key string) (*int64, error)

// GetFormTime safely parses time from form
func GetFormTime(r *http.Request, key string, layout string) (*time.Time, error)

// GetFormString safely gets string from form
func GetFormString(r *http.Request, key string) (*string, error)

// GetFormFileIDs parses comma-separated file IDs
func GetFormFileIDs(r *http.Request, key string) ([]int64, error)
```

### 3. Update Handler Structure

Each handler should:

```go
func New(log *slog.Logger, adder EntityAdder, uploader FileUploader, saver FileMetaSaver) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        contentType := r.Header.Get("Content-Type")

        var req AddEntityRequest
        var fileIDs []int64
        var err error

        if strings.HasPrefix(contentType, "multipart/form-data") {
            // Parse multipart form
            req, err = parseMultipartRequest(r)
            if err != nil {
                // handle error
            }

            // Process uploaded files
            uploadedFileIDs, err := ProcessFormFiles(ctx, r, uploader, saver, categoryID, entityDate)
            if err != nil {
                // handle error with compensation
            }

            // Parse existing file IDs from form
            existingFileIDs, _ := GetFormFileIDs(r, "file_ids")

            // Combine both
            fileIDs = append(existingFileIDs, uploadedFileIDs...)

        } else {
            // Parse JSON (current behavior)
            if err := render.DecodeJSON(r.Body, &req); err != nil {
                // handle error
            }
            fileIDs = req.FileIDs
        }

        // Rest of handler logic (same for both)
        id, err := adder.AddEntity(ctx, req)
        // ...

        if len(fileIDs) > 0 {
            adder.LinkEntityFiles(ctx, id, fileIDs)
        }
    }
}
```

## File Category Mapping

### Entity → Category Mapping

```go
const (
    CategoryIncident  = "incident"
    CategoryDischarge = "discharge"
    CategoryShutdown  = "shutdown"
    CategoryVisit     = "visit"
)
```

Each entity type will use its own category for file organization in MinIO:
- Incidents → `incident/2025/01/15/<uuid>.jpg`
- Discharges → `discharge/2025/01/15/<uuid>.pdf`
- Shutdowns → `shutdown/2025/01/15/<uuid>.jpg`
- Visits → `visit/2025/01/15/<uuid>.pdf`

### Category ID Resolution

Helper function to get category ID by name:

```go
func GetCategoryIDByName(ctx context.Context, repo CategoryGetter, name string) (int64, error)
```

## Error Handling & Compensation

### Upload Compensation Strategy

If entity creation fails after files are uploaded:

```go
// Pseudo-code for compensation
uploadedFileIDs := []int64{}
uploadedObjectKeys := []string{}

// Upload files
for _, file := range files {
    objectKey := generateObjectKey()
    err := uploader.UploadFile(ctx, objectKey, file)
    if err != nil {
        // Rollback: delete all previously uploaded files
        compensateUploads(ctx, uploader, saver, uploadedFileIDs, uploadedObjectKeys)
        return err
    }

    fileID, err := saver.AddFile(ctx, fileModel)
    if err != nil {
        // Rollback
        compensateUploads(ctx, uploader, saver, uploadedFileIDs, uploadedObjectKeys)
        return err
    }

    uploadedFileIDs = append(uploadedFileIDs, fileID)
    uploadedObjectKeys = append(uploadedObjectKeys, objectKey)
}

// Create entity
id, err := adder.AddEntity(ctx, req)
if err != nil {
    // Rollback all file uploads
    compensateUploads(ctx, uploader, saver, uploadedFileIDs, uploadedObjectKeys)
    return err
}

// Success
```

### Compensation Function

```go
func compensateUploads(
    ctx context.Context,
    uploader FileUploader,
    saver FileMetaSaver,
    fileIDs []int64,
    objectKeys []string,
) {
    // Delete from MinIO
    for _, key := range objectKeys {
        uploader.DeleteFile(ctx, key)
    }

    // Delete from DB
    for _, id := range fileIDs {
        saver.DeleteFile(ctx, id)
    }
}
```

## Edit Handler Updates

### Fix Current Issue

Current code only updates files if `len(file_ids) > 0`, making it impossible to remove all files.

**Fix:**
```go
// Check if FileIDs field was explicitly provided (not just non-empty)
if r.Form.Has("file_ids") || req.FileIDs != nil {
    // Unlink old files
    editor.UnlinkEntityFiles(ctx, id)

    // Link new files if any
    if len(fileIDs) > 0 {
        editor.LinkEntityFiles(ctx, id, fileIDs)
    }
}
```

### Edit with Multipart

```http
PATCH /api/incidents/123
Content-Type: multipart/form-data

description=Updated description
files=<new_file_1.jpg>
files=<new_file_2.pdf>
file_ids=45,46  (keep existing files 45 and 46)
```

Behavior:
1. Remove ALL old file links
2. Upload new files (get IDs: 100, 101)
3. Link: [45, 46, 100, 101]

## Response Format

### Success Response (JSON)
```json
{
  "status": "success",
  "id": 123,
  "uploaded_files": [
    {
      "id": 100,
      "file_name": "photo.jpg",
      "size_bytes": 152400
    },
    {
      "id": 101,
      "file_name": "report.pdf",
      "size_bytes": 524288
    }
  ]
}
```

### Success Response (Multipart with files)
```json
{
  "status": "success",
  "id": 123,
  "uploaded_files": [
    {"id": 100, "file_name": "photo.jpg", "size_bytes": 152400}
  ],
  "linked_files": [45, 46]
}
```

## Migration Path

### Phase 1: Add Multipart Support (Non-Breaking)
- Add multipart parsing to handlers
- Keep JSON support intact
- Both work simultaneously

### Phase 2: Frontend Update
- Update frontend to use multipart for file uploads
- Can still use JSON for entities without files

### Phase 3: Deprecation (Optional)
- Consider deprecating standalone file upload endpoint
- Or keep it for bulk file management

## Benefits

1. **Single request**: Upload files + create entity atomically
2. **Better UX**: Simpler frontend code
3. **Transactional safety**: Compensation on failure
4. **Backward compatible**: JSON still works
5. **Flexible**: Mix existing files + new uploads
6. **Cleaner API**: More RESTful design

## Implementation Checklist

- [ ] Create `internal/lib/service/fileupload/helper.go`
- [ ] Create `internal/lib/api/formparser/parser.go`
- [ ] Update discharge add handler
- [ ] Update discharge edit handler
- [ ] Update incident add handler
- [ ] Update incident edit handler
- [ ] Update shutdown add handler
- [ ] Update shutdown edit handler
- [ ] Update visit add handler
- [ ] Update visit edit handler
- [ ] Add tests for multipart parsing
- [ ] Add tests for file upload compensation
- [ ] Update API documentation
- [ ] Update frontend integration

## Example Usage (Frontend)

```javascript
// Create incident with file upload
const formData = new FormData();
formData.append('organization_id', '123');
formData.append('incident_time', '2025-01-15T10:30:00Z');
formData.append('description', 'Water leak');
formData.append('files', file1); // File object
formData.append('files', file2); // File object
formData.append('file_ids', '45,46'); // Existing files

const response = await fetch('/api/incidents', {
  method: 'POST',
  body: formData
});
```
