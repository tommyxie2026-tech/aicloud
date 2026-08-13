package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	RequestIDHeader = "X-Request-ID"
	TraceIDHeader   = "X-Trace-ID"
)

type requestMetadataKey struct{}

type RequestMetadata struct {
	RequestID string
	TraceID   string
}

func WithRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := RequestMetadataFromContext(r.Context())
		if metadata.RequestID == "" {
			metadata.RequestID = normalizeCorrelationID(r.Header.Get(RequestIDHeader))
		}
		if metadata.RequestID == "" {
			metadata.RequestID = newCorrelationID("req")
		}
		if metadata.TraceID == "" {
			metadata.TraceID = normalizeCorrelationID(r.Header.Get(TraceIDHeader))
		}
		if metadata.TraceID == "" {
			metadata.TraceID = newCorrelationID("trace")
		}

		w.Header().Set(RequestIDHeader, metadata.RequestID)
		w.Header().Set(TraceIDHeader, metadata.TraceID)
		r.Header.Set(RequestIDHeader, metadata.RequestID)
		r.Header.Set(TraceIDHeader, metadata.TraceID)
		ctx := context.WithValue(r.Context(), requestMetadataKey{}, metadata)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestMetadataFromContext(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	metadata, _ := ctx.Value(requestMetadataKey{}).(RequestMetadata)
	metadata.RequestID = normalizeCorrelationID(metadata.RequestID)
	metadata.TraceID = normalizeCorrelationID(metadata.TraceID)
	return metadata
}

func normalizeCorrelationID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return ""
	}
	return value
}

func newCorrelationID(prefix string) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return prefix + "-" + hex.EncodeToString(raw[:])
	}
	return prefix + "-unavailable"
}
