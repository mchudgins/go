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
//

package handler

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "httpRequestsReceived_total",
			Help: "Number of HTTP requests received.",
		},
		[]string{"url"},
	)
	httpRequestsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "httpRequestsProcessed_total",
			Help: "Number of HTTP requests processed.",
		},
		[]string{"url", "status"},
	)
	httpRequestDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_duration",
			Help: "Duration of HTTP responses.",
		},
		[]string{"url", "status"},
	)
	httpResponseSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size",
			Help: "Size of http responses",
		},
		[]string{"url"},
	)

	connMapMutex sync.Mutex
	connMap      = make(map[string]func())
	connNew      = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_conn_new",
		Help: "number of new http/tcp connections",
	}, []string{"port"})
	connActive = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_conn_active",
		Help: "number of active http/tcp connections",
	}, []string{"port"})
	connIdle = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_conn_idle",
		Help: "number of idle http/tcp connections",
	}, []string{"port"})
	connClosed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_conn_closed",
		Help: "number of closed http/tcp connections",
	}, []string{"port"})
)

func init() {
	prometheus.MustRegister(httpRequestsReceived)
	prometheus.MustRegister(httpRequestsProcessed)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpResponseSize)
	prometheus.MustRegister(connNew)
	prometheus.MustRegister(connActive)
	prometheus.MustRegister(connIdle)
	prometheus.MustRegister(connClosed)
}

func HTTPMetricsCollector(fn http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		u := r.URL.Path
		httpRequestsReceived.With(prometheus.Labels{
			"url": u,
		}).Inc()

		// we want the status code from the handler chain,
		// so inject an HTTPWriter, if one doesn't exist

		hw, ok := w.(*HTTPWriter)
		if !ok {
			hw = NewHTTPWriter(w)
		}

		// after ServeHTTP runs, collect metrics!

		defer func() {
			status := strconv.Itoa(hw.StatusCode())
			httpRequestsProcessed.With(prometheus.Labels{"url": u, "status": status}).Inc()
			end := time.Now()
			duration := end.Sub(start)
			httpRequestDuration.With(prometheus.Labels{
				"url":    u,
				"status": status,
			}).Observe(float64(duration.Nanoseconds()))
			httpResponseSize.With(prometheus.Labels{
				"url": u,
			}).Observe(float64(hw.Length()))
		}()

		fn.ServeHTTP(hw, r)
	})
}

// HTTPConnectionMetricsCollector generates prometheus metrics for connection state
// see:  https://golang.org/pkg/net/http/#ConnState
func HTTPConnectionMetricsCollector(c net.Conn, newState http.ConnState) {
	addr := c.LocalAddr().String()
	port := addr[strings.LastIndex(addr, ":")+1:]
	remoteAddr := c.RemoteAddr().String()

	//fmt.Printf("HTTPConnectionMetricsCollector: remoteAddr %s; port %s; newState %s\n", remoteAddr, port, newState.String())

	label := prometheus.Labels{"port": port}

	connMapMutex.Lock()
	defer connMapMutex.Unlock()

	switch newState {
	case http.StateNew:
		connNew.With(label).Inc()
		connMap[remoteAddr] = connNew.With(label).Dec

	case http.StateActive:
		connActive.With(label).Inc()
		if dec, ok := connMap[remoteAddr]; ok {
			dec()
		}
		connMap[remoteAddr] = connActive.With(label).Dec

	case http.StateIdle:
		connIdle.With(label).Inc()
		if dec, ok := connMap[remoteAddr]; ok {
			dec()
		}
		connMap[remoteAddr] = connIdle.With(label).Dec

	default: //StateHijacked or StateClosed
		connClosed.With(label).Inc()
		if dec, ok := connMap[remoteAddr]; ok {
			dec()
			delete(connMap, remoteAddr)
		}
	}
}
