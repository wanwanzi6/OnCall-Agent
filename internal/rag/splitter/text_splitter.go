package splitter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/rag"
)

const (
	DefaultChunkSize    = 800
	DefaultChunkOverlap = 100
)

type TextSplitter struct {
	chunkSize    int
	chunkOverlap int
}

func NewTextSplitter(chunkSize, chunkOverlap int) *TextSplitter {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 4
	}
	return &TextSplitter{chunkSize: chunkSize, chunkOverlap: chunkOverlap}
}

func (s *TextSplitter) Split(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	if strings.TrimSpace(doc.Content) == "" {
		return nil, rag.ErrEmptyDocument
	}
	sections := s.sections(doc)
	chunks := make([]domain.Chunk, 0)
	for _, section := range sections {
		for _, content := range s.splitContent(section.content) {
			metadata := copyMetadata(doc.Metadata)
			metadata["source_file"] = sourceFile(doc)
			metadata["title_path"] = section.titlePath
			chunks = append(chunks, domain.Chunk{
				ID:         chunkID(doc.ID, len(chunks), content),
				DocumentID: doc.ID,
				Content:    content,
				Metadata:   metadata,
				Index:      len(chunks),
			})
		}
	}
	if len(chunks) == 0 {
		return nil, rag.ErrEmptyDocument
	}
	return chunks, nil
}

type textSection struct {
	titlePath string
	content   string
}

func (s *TextSplitter) sections(doc domain.Document) []textSection {
	if !isMarkdown(doc) {
		return []textSection{{content: doc.Content}}
	}

	var sections []textSection
	var titleStack [3]string
	var currentTitle string
	var current strings.Builder

	flush := func() {
		content := strings.TrimSpace(current.String())
		if content != "" {
			sections = append(sections, textSection{titlePath: currentTitle, content: content})
		}
		current.Reset()
	}

	for _, line := range strings.Split(doc.Content, "\n") {
		level, title := markdownTitle(line)
		if level > 0 {
			flush()
			titleStack[level-1] = title
			for i := level; i < len(titleStack); i++ {
				titleStack[i] = ""
			}
			currentTitle = joinTitlePath(titleStack)
			current.WriteString(strings.TrimSpace(line))
			current.WriteString("\n")
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	flush()

	if len(sections) == 0 {
		return []textSection{{content: doc.Content}}
	}
	return sections
}

func (s *TextSplitter) splitContent(content string) []string {
	paragraphs := splitParagraphs(content)
	var chunks []string
	var current strings.Builder

	flush := func() {
		text := strings.TrimSpace(current.String())
		if text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
	}

	for _, para := range paragraphs {
		if len(para) > s.chunkSize {
			flush()
			chunks = append(chunks, splitLongText(para, s.chunkSize, s.chunkOverlap)...)
			continue
		}
		nextLen := current.Len() + len(para)
		if current.Len() > 0 {
			nextLen += 2
		}
		if nextLen > s.chunkSize {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}
	flush()
	return chunks
}

func splitParagraphs(content string) []string {
	raw := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n")
	paragraphs := make([]string, 0, len(raw))
	for _, item := range raw {
		text := strings.TrimSpace(item)
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}
	return paragraphs
}

func splitLongText(text string, size, overlap int) []string {
	if size <= 0 {
		size = DefaultChunkSize
	}
	if overlap >= size {
		overlap = 0
	}
	var chunks []string
	for start := 0; start < len(text); {
		end := start + size
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, strings.TrimSpace(text[start:end]))
		if end == len(text) {
			break
		}
		next := end - overlap
		if next <= start {
			next = end
		}
		start = next
	}
	return chunks
}

func markdownTitle(line string) (int, string) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, ""
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level < 1 || level > 3 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, ""
	}
	title := strings.TrimSpace(trimmed[level:])
	if title == "" {
		return 0, ""
	}
	return level, title
}

func joinTitlePath(stack [3]string) string {
	parts := make([]string, 0, len(stack))
	for _, title := range stack {
		if title != "" {
			parts = append(parts, title)
		}
	}
	return strings.Join(parts, " > ")
}

func copyMetadata(input map[string]string) map[string]string {
	output := make(map[string]string, len(input)+2)
	for k, v := range input {
		output[k] = v
	}
	return output
}

func isMarkdown(doc domain.Document) bool {
	ext := strings.ToLower(doc.Metadata["file_ext"])
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(doc.Name))
	}
	return ext == ".md" || ext == ".markdown"
}

func sourceFile(doc domain.Document) string {
	if source := doc.Metadata["source_file"]; source != "" {
		return source
	}
	return doc.Name
}

func chunkID(documentID string, index int, content string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d:%s", documentID, index, content)))
	return "chk_" + hex.EncodeToString(sum[:])[:16]
}
