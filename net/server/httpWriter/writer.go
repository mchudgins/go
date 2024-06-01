package httpWriter

import (
	"net/http"

	"go.uber.org/zap"
)

type HTTPWriter struct {
	w             http.ResponseWriter
	statusCode    int
	contentLength int
	logger        *zap.Logger
}

type Option func(w *HTTPWriter)

func Logger(logger *zap.Logger) Option {
	return func(w *HTTPWriter) { w.logger = logger }
}

func NewHTTPWriter(w http.ResponseWriter, options ...Option) *HTTPWriter {
	writer := &HTTPWriter{w: w}

	for _, option := range options {
		option(writer)
	}

	return writer
}

func (l *HTTPWriter) Header() http.Header {
	return l.w.Header()
}

func (l *HTTPWriter) Write(data []byte) (int, error) {

	if l.logger != nil {
		l.logger.Info("HTTPWriter.Write",
			zap.ByteString("data", data),
			zap.Int("len", len(data)))
	}

	l.contentLength += len(data)
	return l.w.Write(data)
}

func (l *HTTPWriter) WriteHeader(status int) {
	l.statusCode = status
	l.w.WriteHeader(status)
}

func (l *HTTPWriter) Length() int {
	return l.contentLength
}

func (l *HTTPWriter) StatusCode() int {

	// if nobody set the status, but data has been written
	// then all must be well.
	if l.statusCode == 0 && l.contentLength > 0 {
		return http.StatusOK
	}

	return l.statusCode
}
