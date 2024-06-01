// Copyright © 2018 Mike Hudgins <mchudgins@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package handler

import (
	"net/http"

	"go.uber.org/zap"
)

// HTTPWriter wraps a Writer so that the access logger can obtain response headers
// and number of bytes written in the response
type HTTPWriter struct {
	w             http.ResponseWriter
	statusCode    int
	contentLength int
	logger        *zap.Logger
}

// HTTPWriterOption permits customization of an HTTPWriter
type HTTPWriterOption func(w *HTTPWriter)

// EnableLogging permits logging to a zap.Logger of useful
// info during the execution of an HTTPWriter.
func EnableLogging(logger *zap.Logger) HTTPWriterOption {
	return func(w *HTTPWriter) { w.logger = logger }
}

func NewHTTPWriter(w http.ResponseWriter, options ...HTTPWriterOption) *HTTPWriter {
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
		l.logger.Debug("HTTPWriter.Write",
			zap.String("data", string(data)),
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
