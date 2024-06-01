/*
 * Copyright (c) 2024.  Mike Hudgins <mchudgins@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package webapp

import (
	"github.com/justinas/alice"
	"github.com/mchudgins/go/log"
	"github.com/mchudgins/go/net/server/correlationID"
	"github.com/mchudgins/go/net/server/healthcheck"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"net/http"
)

func (s *WebApp) routes() {
	s.router.NotFoundHandler = notFoundHandler()
	s.router.MethodNotAllowedHandler = methodNotAllowedHandler()

	s.chain = s.chain.Append(s.contextLogger(), rateLimit(10, 50))

	// health checks
	health := healthcheck.NewHandler()
	health.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(25))

	s.router.Handle(
		"/healthz",
		healthcheck.HealthCheckAPI(),
	).
		Methods(http.MethodGet).
		Name("health check handler")

	// make prom metrics available
	s.router.Handle(
		"/metrics",
		promhttp.Handler(),
	).
		Methods(http.MethodGet).
		Name("prometheus metrics handler")

}

// contextLogger adds the per-request fields we care about to each log message
func (s *WebApp) contextLogger() alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxLogger := s.logger.With(zap.String(correlationID.RequestIDKey, correlationID.FromContext(ctx)))

			ctx = log.NewContext(ctx, ctxLogger)
			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}

func rateLimit(limit rate.Limit, burst int) alice.Constructor {
	rl := rate.NewLimiter(limit, burst)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl.Allow() {
				h.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusTooManyRequests)
			}
		})
	}
}

// notFoundHandler
func notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
}

// methodNotAllowedHandler
func methodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
