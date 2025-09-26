package file

import "time"

type LatestFile struct {
	ID           int64     `json:"id"`
	FileName     string    `json:"file_name"`
	ObjectKey    string    `json:"object_key"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
}
