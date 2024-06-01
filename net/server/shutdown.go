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

package server

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
)

/*
	Handle the graceful shutdown of the server endpoints
*/

type sourcetype int

const (
	interrupt sourcetype = iota
	httpServer
	metricsServer
	rpcServer
	unknown
)

type eventSource struct {
	source sourcetype
	err    error
}

func (t sourcetype) String() string {
	sourcetypeNames := []string{"interrupt", "httpServer", "metricServer", "rpcServer", "unknown"}

	return sourcetypeNames[t]
}

func (cfg *Config) performGracefulShutdown(errc chan eventSource, evtSrc eventSource) {
	cfg.logger.Info("termination event detected", zap.Error(evtSrc.err), zap.String("source", evtSrc.source.String()))
	waitDuration := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), waitDuration)
	defer cancel()

	waitEvents := 0

	if evtSrc.source != httpServer && cfg.httpServer != nil {
		waitEvents++
		go func() {
			if err := cfg.httpServer.Shutdown(ctx); err != nil {
				cfg.logger.Error("httpServer.Shutdown", zap.Error(err))

				//				if cfg.wg != nil {
				//					cfg.wg.Add(-1) // she's not going down on her own...
				//				}

				// if we're here (in performGracefulShutdown), don't re-initiate shutdown
				// by sending on the errc chan !?
				//errc <- eventSource{
				//	err:    err,
				//	source: httpServer,
				//}
			}
		}()
	}
	if evtSrc.source != rpcServer && cfg.rpcServer != nil {
		waitEvents++
		go func() {
			cfg.rpcServer.GracefulStop()
		}()
	}
	if evtSrc.source != metricsServer && cfg.metricsServer != nil {
		waitEvents++
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), waitDuration)
			defer cancel()

			if err := cfg.metricsServer.Shutdown(ctx); err != nil {
				cfg.logger.Error("metricsServer.Shutdown", zap.Error(err))

				if cfg.wg != nil {
					cfg.wg.Add(-1) // she's not going down on her own...
				}

				errc <- eventSource{
					err:    err,
					source: metricsServer,
				}
			}
		}()
	}

	// wait for shutdown to complete or time to expire
	for waitEvents > 0 {
		select {
		case <-time.After(waitDuration + 1*time.Second):
			cfg.logger.Info("server shutdown complete")
			os.Exit(1)

		case <-ctx.Done():
			cfg.logger.Warn("wait time for service shutdown has elapsed -- performing hard shutdown", zap.Error(ctx.Err()))
			os.Exit(2)

		case evt := <-errc:
			waitEvents--
			cfg.logger.Info("listener shutdown notice recv'ed", zap.Error(evt.err), zap.String("eventSource", evt.source.String()))
			cfg.logger.Debug("listener shutdown", zap.Int("waitEvents", waitEvents))
		}
	}

	//	os.Exit(0)
}
