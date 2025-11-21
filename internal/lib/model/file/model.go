package file

import "time"

type Model struct {
	ID         int64     `json:"id"`
	FileName   string    `json:"file_name"`
	ObjectKey  string    `json:"-"` // Ключ в MinIO — внутренняя информация
	CategoryID int64     `json:"category_id"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
}
