package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

const HeaderName = "X-Trace-ID"

type contextKey struct{}

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		traceID = NewID()
	}
	return context.WithValue(ctx, contextKey{}, traceID)
}

func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(contextKey{}).(string); ok {
		return traceID
	}
	return ""
}
