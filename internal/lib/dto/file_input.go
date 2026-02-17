package dto

import "io"

// FileInput abstracts a file received from the HTTP layer (multipart form)
// so that service-layer code has no dependency on net/http.
type FileInput struct {
	Filename    string
	Size        int64
	ContentType string
	Open        func() (io.ReadCloser, error)
}
