package dto

import "time"

// FileResponse represents a file in API responses with a presigned URL for download.
type FileResponse struct {
	ID         int64     `json:"id"`
	FileName   string    `json:"file_name"`
	CategoryID int64     `json:"category_id"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
	URL        string    `json:"url"` // Presigned URL for file download (1 hour expiration)
}
