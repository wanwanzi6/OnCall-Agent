package domain

type ChatResult struct {
	TraceID    string           `json:"trace_id,omitempty"`
	Answer     string           `json:"answer"`
	Sources    []string         `json:"sources"`
	Citations  []Citation       `json:"citations,omitempty"`
	Mock       bool             `json:"mock"`
	Plan       *AgentPlan       `json:"plan,omitempty"`
	Iterations []AgentIteration `json:"iterations,omitempty"`
	Steps      []WorkflowStep   `json:"steps,omitempty"`
}

type Citation struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Source     string  `json:"source"`
	Score      float64 `json:"score"`
	Content    string  `json:"content"`
}

type StreamChunk struct {
	Index int    `json:"index"`
	Delta string `json:"delta"`
	Done  bool   `json:"done"`
}
