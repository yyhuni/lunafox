package transport

import (
	"context"
	"net/http"
)

// responseObserver captures only status and byte count; it never retains a
// response body because MCP audit records are intentionally payload-free.
type responseObserver struct {
	http.ResponseWriter
	ctx    context.Context
	status int
	bytes  int
}

func (observer *responseObserver) WriteHeader(status int) {
	if observer.status != 0 || observer.cancelled() != nil {
		return
	}
	observer.status = status
	observer.ResponseWriter.WriteHeader(status)
}

func (observer *responseObserver) Write(data []byte) (int, error) {
	if err := observer.cancelled(); err != nil {
		return 0, err
	}
	if observer.status == 0 {
		observer.WriteHeader(http.StatusOK)
	}
	written, err := observer.ResponseWriter.Write(data)
	observer.bytes += written
	return written, err
}

func (observer *responseObserver) Unwrap() http.ResponseWriter { return observer.ResponseWriter }

func (observer *responseObserver) Flush() {
	if observer.cancelled() != nil {
		return
	}
	if flusher, ok := observer.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (observer *responseObserver) cancelled() error {
	if observer.ctx == nil {
		return nil
	}
	return observer.ctx.Err()
}
