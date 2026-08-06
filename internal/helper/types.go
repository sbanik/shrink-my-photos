package helper

type FileRecord struct {
	OriginalPath string `json:"original_path"`
	// StagedPath is retained only to read manifests created by versions before
	// in-place processing. New records leave it empty.
	StagedPath    string `json:"staged_path,omitempty"`
	DiscardedPath string `json:"discarded_path,omitempty"`
	WebPPath      string `json:"webp_path"`
	OriginalSize  int64  `json:"original_size_bytes"`
	WebPSize      int64  `json:"webp_size_bytes"`
	Status        string `json:"status"`
}

type Manifest struct {
	Records     map[string]*FileRecord   `json:"records"`
	HiddenFiles map[string]*HiddenRecord `json:"hidden_files"`
}

type HiddenRecord struct {
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	Status string `json:"status"`
}
