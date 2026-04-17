package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type UploadInitRequest struct {
	Filename    string `json:"filename" validate:"required"`
	FileSize    int64  `json:"filesize" validate:"required,min=1"`
	ContentType string `json:"contentType" validate:"required"`
}

type UploadCompletePart struct {
	PartNumber int32  `json:"partNumber"`
	ETag       string `json:"etag"`
}

type UploadCompleteRequest struct {
	Salt  string               `json:"salt" validate:"required"`
	Parts []UploadCompletePart `json:"parts"`
}

type UploadAbortRequest struct {
	Salt string `json:"salt" validate:"required"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

type UploadSmallResponse struct {
	PresignedURL string `json:"presignedUrl"`
	Salt         string `json:"salt"`
	Key          string `json:"key"`
}

type UploadLargePart struct {
	PartNumber   int    `json:"partNumber"`
	PresignedURL string `json:"presignedUrl"`
}

type UploadLargeResponse struct {
	UploadID string            `json:"uploadId"`
	Salt     string            `json:"salt"`
	Key      string            `json:"key"`
	Parts    []UploadLargePart `json:"parts"`
}

type UploadCompleteResponse struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}
