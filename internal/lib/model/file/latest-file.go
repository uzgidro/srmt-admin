package file

import (
	"path/filepath"
	"time"
)

type LatestFile struct {
	ID           int64     `json:"id"`
	FileName     string    `json:"file_name"`
	ObjectKey    string    `json:"-"`
	SizeBytes    int64     `json:"size_bytes"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
}

func (lf *LatestFile) GetExtension() string {
	return filepath.Ext(lf.FileName)
}
