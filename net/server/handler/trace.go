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
	"time"

	//zlog "github.com/opentracing/opentracing-go/log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	serviceHostPort    = "localhost:8080"
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"
)

type traceConfig struct {
	batchInterval time.Duration
	batchSize     int
	debug         bool
	logger        *zap.Logger
}

func (t *traceConfig) Log(keyval ...interface{}) error {
	if t == nil {
		return nil
	}

	len := len(keyval)
	fields := make([]zapcore.Field, len/2+1)

	for i := 0; i < len; i = +2 {
		if key, ok := keyval[i].(string); ok {
			switch keyval[i+1].(type) {
			case string:
				fields = append(fields, zap.String(key, keyval[i+1].(string)))

			case int:
				fields = append(fields, zap.Int(key, keyval[i+1].(int)))

			case []byte:
				fields = append(fields, zap.ByteString(key, keyval[i+1].([]byte)))

			case error:
				fields = append(fields, zap.Error(keyval[i+1].(error)))

			case time.Duration:
				fields = append(fields, zap.Duration(key, keyval[i+1].(time.Duration)))

			case time.Time:
				fields = append(fields, zap.Time(key, keyval[i+1].(time.Time)))

			case bool:
				fields = append(fields, zap.Bool(key, keyval[i+1].(bool)))

			default:
				fields = append(fields, zap.Any(key, keyval[i+1]))
			}
		} else {
			t.logger.Warn("key name is not of type 'string'",
				zap.Any("key", key))
		}
	}

	t.logger.Debug("opentrace-log-event", fields...)

	return nil
}

// TraceOption permits customization of an Tracer
type TracerOption func(t *traceConfig)

// Logger permits logging to a zap.Logger of useful
// info during the execution of a tracer.
func Logger(logger *zap.Logger) TracerOption {
	return func(t *traceConfig) { t.logger = logger }
}

func BatchInterval(d time.Duration) TracerOption {
	return func(t *traceConfig) { t.batchInterval = d }
}

func BatchSize(size int) TracerOption {
	return func(t *traceConfig) { t.batchSize = size }
}

func DebugMode() TracerOption {
	return func(t *traceConfig) { t.debug = true }
}

/*  mch -- 2019-11-07 new zipkin-go repo no longer matches these API's
    and will need a MAJOR overhaul.  commented out until that day....
func NewTracer(serviceName string, options ...TracerOption) (*opentracing.Tracer, error) {

	// default config for the tracer/collector
	cfg := &traceConfig{
		batchInterval: 5 * time.Second,
		batchSize:     100,
	}

	// process callers desired changes in the default config
	for _, o := range options {
		o(cfg)
	}

	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint,
		zipkin.HTTPLogger(cfg),
		zipkin.HTTPClient(hystrix.NewClient("zipkin")),
		zipkin.HTTPBatchInterval(cfg.batchInterval),
		zipkin.HTTPBatchSize(cfg.batchSize))
	if err != nil {
		return nil, fmt.Errorf("unable to create NewHTTPCollector -- %v", err.Error())
	}

	tracer, err := openzipkin.NewTracer(
		openzipkin.NewRecorder(collector, debugMode, serviceHostPort, serviceName),
		openzipkin.WithLogger(cfg),
		openzipkin.DebugMode(cfg.debug),
		//		zipkin.ClientServerSameSpan(true),
	)

	if err != nil {
		return nil, fmt.Errorf("unable to create NewTracer -- %v", err.Error())
	}

	opentracing.SetGlobalTracer(tracer)

	return &tracer, nil
}

*/

// HandlerFunc is a middleware function for incoming HTTP requests.
type HandlerFunc func(next http.Handler) http.Handler

// FromHTTPRequest returns a Middleware HandlerFunc that tries to join with an
// OpenTracing trace found in the HTTP request headers and starts a new Span
// called `operationName`. If no trace could be found in the HTTP request
// headers, the Span will be a trace root. The Span is incorporated in the
// HTTP Context object and can be retrieved with
// opentracing.SpanFromContext(ctx).

/*  this is legacy.  should be deleted.
func TracerFromHTTPRequest(tracer opentracing.Tracer, operationName string,
) HandlerFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			span, ctx := opentracing.StartSpanFromContext(req.Context(), operationName)
			defer span.Finish()

			// tag this request with a correlation ID, so we can troubleshoot it later, if necessary
			req, corrID := correlationID.FromRequest(req)
			w.Header().Set(correlationID.CORRID, corrID)
			span.SetTag(correlationID.CORRID, corrID)
			ext.HTTPUrl.Set(span, req.URL.Path)

			// store span in context
			ctx = opentracing.ContextWithSpan(req.Context(), span)

			// update request context to include our new span
			req = req.WithContext(ctx)

			// we want the status code from the handler chain,
			// so inject an HTTPWriter, if one doesn't exist

			if _, ok := w.(*httpWriter.HTTPWriter); !ok {
				w = httpWriter.NewHTTPWriter(w)
			}

			// next middleware or actual request handler
			next.ServeHTTP(w, req)

			if hw, ok := w.(*httpWriter.HTTPWriter); ok {
				span.SetTag(string(ext.HTTPStatusCode), hw.StatusCode())
			}
		})
	}
}
*/

/*  mch -- 2019-11-07 new zipkin-go repo no longer matches these API's
    and will need a MAJOR overhaul.  commented out until that day....
func TracerFromHTTPRequest(tracer *opentracing.Tracer, operationName string,
) HandlerFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

			// tag this request with a correlation ID, so we can troubleshoot it later, if necessary
			corrID, fExisted := correlationID.FromRequest(req)
			if !fExisted {
				corrID = correlationID.NewID()
				ctx := correlationID.NewContext(req.Context(), corrID)
				req = req.WithContext(ctx)
			}

			// if we're at the edge of the system, send the correlation ID back in the response
			if !fExisted {
				w.Header().Set(correlationID.CORRID, corrID)
			}

			var serverSpan opentracing.Span
			//			appSpecificOperationName := operationName
			appSpecificOperationName := req.Method + ":" + req.URL.Path
			wireContext, err := opentracing.GlobalTracer().Extract(
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(req.Header))
			if err != nil {
				// no need to handle, we'll just create a parent span later
				//log.WithError(err).Error("unable to extract wire context")
			}

			// Create the span referring to the RPC client if available.
			// If wireContext == nil, a root span will be created.
			serverSpan = opentracing.StartSpan(
				appSpecificOperationName,
				ext.RPCServerOption(wireContext))

			defer serverSpan.Finish()

			ext.HTTPUrl.Set(serverSpan, req.URL.Path)
			serverSpan.SetTag(correlationID.CORRID, corrID)

			//
			//	serverSpan.LogFields(
			//		zlog.String(string(ext.HTTPUrl), req.URL.Path),
			//	)
			//

			ctx := opentracing.ContextWithSpan(req.Context(), serverSpan)

			// update request context to include our new span
			req = req.WithContext(ctx)

			// we want the status code from the handler chain,
			// so inject an HTTPWriter, if one doesn't exist

			var hw *HTTPWriter
			hw, ok := w.(*HTTPWriter)
			if !ok {
				hw = NewHTTPWriter(w)
			}
			defer func() {
				serverSpan.SetTag(string(ext.HTTPStatusCode), hw.StatusCode())
			}()

			// next middleware or actual request handler
			next.ServeHTTP(hw, req)
		})
	}
}
*/
