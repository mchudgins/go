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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// handlerWithContext is a handler implementation supporting context.Context.
type handlerWithContext struct {
	checksMutex     sync.RWMutex
	livenessChecks  map[string]CheckWithContext
	readinessChecks map[string]CheckWithContext
}

func NewHandler() Handler {
	h := &handlerWithContext{
		livenessChecks:  make(map[string]CheckWithContext),
		readinessChecks: make(map[string]CheckWithContext),
	}
	//h.Handle("/live", http.HandlerFunc(h.LiveEndpoint))
	//h.Handle("/ready", http.HandlerFunc(h.ReadyEndpoint))

	return h
}

func HealthCheckAPI() http.Handler {
	h := NewHandler()

	h.AddLivenessCheck("goroutine-threshold", GoroutineCountCheck(25))

	return h
}

func (s *handlerWithContext) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/live") {
		s.LiveEndpoint(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/ready") {
		s.ReadyEndpoint(w, r)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprintf(w, "valid health check endpoints are /healthz/live and /healthz/ready\n")
}

func (s *handlerWithContext) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.livenessChecks)
}

func (s *handlerWithContext) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.readinessChecks, s.livenessChecks)
}

func (s *handlerWithContext) AddLivenessCheck(name string, check CheckWithContext) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *handlerWithContext) AddReadinessCheck(name string, check CheckWithContext) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *handlerWithContext) collectChecks(ctx context.Context, checks map[string]CheckWithContext, resultsOut map[string]string, statusOut *int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()
	for name, check := range checks {
		if err := check(ctx); err != nil {
			*statusOut = http.StatusServiceUnavailable
			resultsOut[name] = err.Error()
		} else {
			resultsOut[name] = "OK"
		}
	}
}

func (s *handlerWithContext) handle(w http.ResponseWriter, r *http.Request, checks ...map[string]CheckWithContext) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkResults := make(map[string]string)
	status := http.StatusOK
	for _, check := range checks {
		s.collectChecks(r.Context(), check, checkResults, &status)
	}

	// write out the response code and content type header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// unless ?full=1, return an empty body. Kubernetes only cares about the
	// HTTP status code, so we won't waste bytes on the full body.
	if r.URL.Query().Get("full") != "1" {
		_, _ = w.Write([]byte("{}\n"))
		return
	}

	// otherwise, write the JSON body ignoring any encoding errors (which
	// shouldn't really be possible since we're encoding a map[string]string).
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	_ = encoder.Encode(checkResults)
}
