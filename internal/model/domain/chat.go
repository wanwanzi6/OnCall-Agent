package domain

type ChatResult struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"`
	Mock    bool     `json:"mock"`
}

type StreamChunk struct {
	Index int    `json:"index"`
	Delta string `json:"delta"`
	Done  bool   `json:"done"`
}
