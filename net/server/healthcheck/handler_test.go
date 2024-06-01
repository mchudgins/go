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

package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		live       bool
		ready      bool
		expect     int
		expectBody string
	}{
		{
			name:   "GET /foo should generate a 404",
			method: "POST",
			path:   "/foo",
			live:   true,
			ready:  true,
			expect: http.StatusNotFound,
		},
		{
			name:   "POST /live should generate a 405 Method Not Allowed",
			method: "POST",
			path:   "/live",
			live:   true,
			ready:  true,
			expect: http.StatusMethodNotAllowed,
		},
		{
			name:   "POST /ready should generate a 405 Method Not Allowed",
			method: "POST",
			path:   "/ready",
			live:   true,
			ready:  true,
			expect: http.StatusMethodNotAllowed,
		},
		{
			name:       "with no checks, /live should succeed",
			method:     "GET",
			path:       "/live",
			live:       true,
			ready:      true,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with no checks, /ready should succeed",
			method:     "GET",
			path:       "/ready",
			live:       true,
			ready:      true,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with a failing readiness check, /live should still succeed",
			method:     "GET",
			path:       "/live?full=1",
			live:       true,
			ready:      false,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with a failing readiness check, /ready should fail",
			method:     "GET",
			path:       "/ready?full=1",
			live:       true,
			ready:      false,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-readiness-check\": \"failed readiness check\"\n}\n",
		},
		{
			name:       "with a failing liveness check, /live should fail",
			method:     "GET",
			path:       "/live?full=1",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-liveness-check\": \"failed liveness check\"\n}\n",
		},
		{
			name:       "with a failing liveness check, /ready should fail",
			method:     "GET",
			path:       "/ready?full=1",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-liveness-check\": \"failed liveness check\"\n}\n",
		},
		{
			name:       "with a failing liveness check, /ready without full=1 should fail with an empty body",
			method:     "GET",
			path:       "/ready",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler()

			if !tt.live {
				h.AddLivenessCheck("test-liveness-check", func(context.Context) error {
					return errors.New("failed liveness check")
				})
			}

			if !tt.ready {
				h.AddReadinessCheck("test-readiness-check", func(context.Context) error {
					return errors.New("failed readiness check")
				})
			}

			req, err := http.NewRequest(tt.method, tt.path, nil)
			assert.NoError(t, err)

			reqStr := tt.method + " " + tt.path
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			assert.Equal(t, tt.expect, rr.Code, "wrong code for %q", reqStr)

			if tt.expectBody != "" {
				assert.Equal(t, tt.expectBody, rr.Body.String(), "wrong body for %q", reqStr)
			}
		})
	}
}
