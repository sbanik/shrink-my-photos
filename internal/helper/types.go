package helper

type FileRecord struct {
	OriginalPath string `json:"original_path"`
	StagedPath   string `json:"staged_path"`
	WebPPath     string `json:"webp_path"`
	OriginalSize int64  `json:"original_size_bytes"`
	WebPSize     int64  `json:"webp_size_bytes"`
	Status       string `json:"status"`
}

type Manifest struct {
	Records map[string]*FileRecord `json:"records"`
}