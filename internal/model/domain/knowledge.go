package domain

type UploadResult struct {
	FileName   string   `json:"file_name"`
	FileType   string   `json:"file_type"`
	ChunkCount int      `json:"chunk_count"`
	DocID      string   `json:"doc_id"`
	NextSteps  []string `json:"next_steps"`
	Mock       bool     `json:"mock"`
}
