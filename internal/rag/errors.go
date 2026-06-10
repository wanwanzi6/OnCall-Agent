package rag

import "errors"

var (
	ErrEmptyDocument       = errors.New("document is empty")
	ErrUnsupportedDocument = errors.New("unsupported document type")
	ErrMissingDocumentPath = errors.New("document path is required")
)
