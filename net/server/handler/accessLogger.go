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
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/mchudgins/go/log"
	"github.com/mchudgins/go/net/server/correlationID"
	"github.com/mchudgins/go/net/server/requestTS"
	"github.com/mchudgins/go/net/server/user"
)

func rpcClientInfo(ctx context.Context) (string, string, error) {

	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "unauthenticated")
	}
	clientIP := p.Addr.String()

	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", clientIP, status.Error(codes.Unauthenticated, "unexpected peer transport credentials")
	}

	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		return "", clientIP, status.Error(codes.Unauthenticated, "could not verify peer certificate")
	}

	return strings.ToLower(tlsAuth.State.VerifiedChains[0][0].Subject.CommonName),
		strings.ToLower(clientIP),
		nil
}

func RPCEndpointLog(logger *zap.Logger, s string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		start := time.Now()

		mdIn, okIn := metadata.FromIncomingContext(ctx)
		remoteUser, remoteAddr, _ := rpcClientInfo(ctx)

		// ensure a correlation ID exists
		var corrID string
		var corrHdr = strings.ToLower(correlationID.CORRID) // metadata uses lowercase keys
		if okIn && len(mdIn[corrHdr]) == 1 {
			corrID = mdIn[corrHdr][0]
		} else {
			corrID = correlationID.NewID()
			mdIn.Append(corrHdr, corrID)
			ctx = metadata.NewIncomingContext(ctx, mdIn)
		}
		// add the corrID to the context as well
		ctx = correlationID.NewContext(ctx, corrID)

		// grpc.SendHeader(ctx, metadata.Pairs(correlationID.CORRID, corrID))

		fields := make([]zapcore.Field, 0, 24+len(mdIn))
		if len(s) > 0 {
			fields = append(fields, zap.String("service", s))
		}
		fields = append(fields, zap.String("method", info.FullMethod))
		fields = append(fields, zap.String("remoteIP", remoteAddr))
		if len(remoteUser) > 0 {
			fields = append(fields, zap.String("remoteUser", remoteUser))
		}
		fields = append(fields, zap.String(correlationID.RequestIDKey, corrID))
		if okIn {
			fields = append(fields, zap.Any("requestHeaders", mdIn))
		}

		ctx = log.NewContext(ctx,
			logger.With(
				zap.String("requestID", corrID),
			))
		// tag this request with a timestamp, so we can correlate it via the timestamp
		ctx = requestTS.NewContext(ctx, start)

		defer func() {
			mdOut, okOut := metadata.FromOutgoingContext(ctx)

			end := time.Now()
			elapsed := end.UnixMilli() - start.UnixMilli() // float64(end.Sub(start).Nanoseconds()) / 1000.0 // microSeconds
			grpc.SetTrailer(ctx, metadata.Pairs("duration",
				strconv.FormatInt(elapsed, 10),
				correlationID.CORRID, corrID))
			fields = append(fields, zap.Int64("duration", elapsed))
			fields = append(fields, zap.String("time", start.Format("20060102030405.000000")))
			if okOut {
				fields = append(fields, zap.Any("responseHeaders", mdOut))
			}

			logger.Info("rpc-request", fields...)
		}()

		rc, err := handler(ctx, req)
		if err != nil {
			fields = append(fields, zap.Error(err))
		}
		fields = append(fields, zap.Uint32("status", uint32(status.Code(err))))

		return rc, err
	}
}

func getRequestURIFromRaw(rawURI string) string {
	if !strings.Contains(rawURI, "?") {
		return rawURI
	}

	i := strings.Index(rawURI, "?")

	return rawURI[:i]
}

// HTTPAccessLogger returns a 'func(http.Handler) http.Handler' which
// logs details about the request using a zap.Logger.
//
// It is intended to be used as part of an alice.chain() where
// multiple handlers, acting as 'middleware', wrap a sequence of
// handlers, e.g.,
//
//	chain := alice.Chain( handler1, handler2, HTTPAccessLogger(logger), handler4,...)
//
// Note: If you want to use something other than zap, then simply write
// a different http.Handler!
func HTTPAccessLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			// tag this request with a correlation ID, so we can troubleshoot it later, if necessary
			corrID, fExisted := correlationID.FromRequest(r)
			if !fExisted {
				corrID = correlationID.NewID()
				ctx := correlationID.NewContext(r.Context(), corrID)
				r = r.WithContext(ctx)
			} else { // ensure the correlationID is part of the request Context
				if len(correlationID.FromContext(r.Context())) == 0 {
					ctx := correlationID.NewContext(r.Context(), corrID)
					r = r.WithContext(ctx)
				}
			}

			// tag this request with a timestamp, so we can correlate it via the timestamp
			r = r.WithContext(requestTS.NewContext(r.Context(), start))

			// we want the status code from the handler chain,
			// so inject an HTTPWriter, if one doesn't exist
			lw, ok := w.(*HTTPWriter)
			if !ok {
				lw = NewHTTPWriter(w)
			}

			// ensure the caller gets a correlation ID in the response
			lw.Header().Set(correlationID.CORRID, corrID)

			// save some values, in case the handler changes 'em
			host := r.Host
			url := getRequestURIFromRaw(r.RequestURI)
			remoteAddr := r.RemoteAddr
			method := r.Method
			proto := r.Proto

			fields := make([]zapcore.Field, 0, 20+len(r.Header))
			requestHeaders := make(map[string]string)
			for key := range r.Header {
				requestHeaders[key] = r.Header.Get(key)
			}

			fields = append(fields, zap.String("Host", host))
			fields = append(fields, zap.String("URL", url))
			fields = append(fields, zap.String("remoteIP", remoteAddr))
			fields = append(fields, zap.String("method", method))
			fields = append(fields, zap.String("proto", proto))
			fields = append(fields, zap.Any("requestHeaders", requestHeaders))
			fields = append(fields, zap.String(correlationID.RequestIDKey, corrID))

			defer func() {
				fields = append(fields, zap.Int("status", lw.StatusCode()))
				fields = append(fields, zap.Int("length", lw.Length()))

				// maybe the X-Request-ID was set on the way back?
				//				idIn := r.Header.Get(correlationID.CORRID)
				//				idOut := lw.Header().Get(correlationID.CORRID)
				//				if len(idIn) == 0 && len(idOut) > 0 {
				//					fields = append(fields, zap.String(correlationID.CORRID, idOut))
				//				}

				responseHeaders := make(map[string]string)
				for key := range lw.Header() {
					if key == correlationID.CORRID {
						continue // tracking this header as a separate field in the parent struct
					}
					responseHeaders[key] = lw.Header().Get(key)
				}
				fields = append(fields, zap.Any("responseHeaders", responseHeaders))

				end := time.Now()
				elapsed := float64(end.UnixMilli() - start.UnixMilli()) // microSeconds

				fields = append(fields, zap.Float64("duration", elapsed))
				fields = append(fields, zap.String("time", start.Format("20060102030405.000000")))

				// who dat? Not all requests use X-Remote-User to xmit userid/username
				// so look in the request context if X-Remote-User was not populated.
				uid := user.FromContext(r.Context())
				if len(uid) > 0 {
					fields = append(fields, zap.String("user", uid))
				}
				log.With(fields...).Info("http-request")
			}()

			h.ServeHTTP(lw, r)

		})
	}
}
