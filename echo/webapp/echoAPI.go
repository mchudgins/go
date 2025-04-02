/*
 * Copyright © 2025.  Mike Hudgins <mchudgins@gmail.com>
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

package webapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/gddo/httputil"
	"github.com/mchudgins/go/log"
	"go.uber.org/zap"
)

func (s *Server) echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := log.FromContext(r.Context())

		// figure out the format used to serialize the response
		preferredContentType := httputil.NegotiateContentType(
			r,
			[]string{"text/plain", "application/json", "application/x-protobuf"},
			"application/json",
		)

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		type responseStruct struct {
			Environment []string `json:"environment"`
			Hostname    string   `json:"hostname"`
		}
		response := &responseStruct{
			Environment: os.Environ(),
			Hostname:    hostname,
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Vary", "Accept,Accept-Encoding")

		switch preferredContentType {
		case "text/plain":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Environment\n")
			for _, v := range response.Environment {
				fmt.Fprintf(w, "\t%s\n", v)
			}
			fmt.Fprintf(w, "\nHostname\n")
			fmt.Fprintf(w, "\t%s\n", hostname)

		case "application/json":
			jw := json.NewEncoder(w)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			err := jw.Encode(response)
			if err != nil {
				logger.Warn("unable to encode reponse",
					zap.Error(err))
			}

		default:
			logger.Info("unknown preferredContentType; expected text/plain as default",
				zap.String("preferredContentType", preferredContentType),
				zap.String("Accept", r.Header.Get("Accept")))
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
