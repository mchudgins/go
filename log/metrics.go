/*
 * Copyright © 2022.  Mike Hudgins <mchudgins@gmail.com>
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 */

package log

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap/zapcore"
)

var (
	debugMsgCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "logging_debug_msgs_total",
			Help: "Number of debug messages logged.",
		},
	)
	infoMsgCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "logging_info_msgs_total",
			Help: "Number of informational messages logged.",
		},
	)
	warnMsgCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "logging_warn_msgs_total",
			Help: "Number of warning messages logged.",
		},
	)
	errorMsgCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "logging_error_msgs_total",
			Help: "Number of error messages logged.",
		},
	)
)

func PrometheusMetrics(e zapcore.Entry) error {
	switch e.Level {
	case zapcore.DebugLevel:
		debugMsgCount.Inc()

	case zapcore.InfoLevel:
		infoMsgCount.Inc()

	case zapcore.WarnLevel:
		warnMsgCount.Inc()

	case zapcore.ErrorLevel:
		warnMsgCount.Inc()

	default:
	}

	return nil
}

func init() {
	prometheus.MustRegister(debugMsgCount)
	prometheus.MustRegister(infoMsgCount)
	prometheus.MustRegister(warnMsgCount)
	prometheus.MustRegister(errorMsgCount)
}
