package domain

import "time"

type Document struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Path      string            `json:"path,omitempty"`
	Content   string            `json:"content,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type Chunk struct {
	ID         string            `json:"id"`
	DocumentID string            `json:"document_id"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Index      int               `json:"index"`
}

type SearchResult struct {
	Chunk     Chunk   `json:"chunk"`
	Score     float64 `json:"score"`
	Source    string  `json:"source,omitempty"`
	TitlePath string  `json:"title_path,omitempty"`
}

type IndexResult struct {
	DocumentID string `json:"document_id"`
	ChunkCount int    `json:"chunk_count"`
}

type UploadResult struct {
	TraceID    string           `json:"trace_id,omitempty"`
	FileName   string           `json:"file_name"`
	FileType   string           `json:"file_type"`
	ChunkCount int              `json:"chunk_count"`
	DocID      string           `json:"doc_id"`
	NextSteps  []string         `json:"next_steps"`
	Mock       bool             `json:"mock"`
	Plan       *AgentPlan       `json:"plan,omitempty"`
	Iterations []AgentIteration `json:"iterations,omitempty"`
	Steps      []WorkflowStep   `json:"steps,omitempty"`
}

type KnowledgeSearchResult struct {
	TraceID    string           `json:"trace_id,omitempty"`
	Results    []SearchResult   `json:"results"`
	Plan       *AgentPlan       `json:"plan,omitempty"`
	Iterations []AgentIteration `json:"iterations,omitempty"`
	Steps      []WorkflowStep   `json:"steps,omitempty"`
}
