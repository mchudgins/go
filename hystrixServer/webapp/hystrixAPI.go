/*
 * Copyright (c) 2025.  Mike Hudgins <mchudgins@gmail.com>
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
	"encoding/json"
	"github.com/golang/gddo/httputil"
	"github.com/mchudgins/go/log"
	"go.uber.org/zap"
	"math"
	"math/rand"
	"net/http"
)

type HystrixGaugeInfo struct {
	ServiceName                   string
	CircuitClosed                 bool
	RequestRatePerHost            float32
	RequestRatePerCluster         float32
	HostsInCluster                int
	MedianResponseTimeMS          int
	MeanResponseTimeMS            int
	ResponseTime90thPercentileMS  int
	ResponseTime99thPercentileMS  int
	ResponseTime995thPercentileMS int
	// 10 second counters with 1 second granularity
	SuccessfulRequests     int
	ShortCircuitedRequests int
	ThreadTimeouts         int
	ThreadPoolRejections   int
	Failures               int
	ErrorPercentage        float32
}

const response0 = `[
{ "name": "fubarA", "msg": "mars", "id": 0 },
{ "name": "fubarB", "msg": "venus", "id": 1 }
]
`

const response1 = `[
{ "name": "fubarA", "msg": "saturn", "id": 0 },
{ "name": "fubarB", "msg": "jupiter", "id": 1 }
]
`

var response = []HystrixGaugeInfo{
	{
		ServiceName:                   "LocoWebFrontEnd",
		CircuitClosed:                 true,
		RequestRatePerHost:            1,
		RequestRatePerCluster:         2,
		HostsInCluster:                2,
		MedianResponseTimeMS:          40,
		MeanResponseTimeMS:            55,
		ResponseTime90thPercentileMS:  60,
		ResponseTime99thPercentileMS:  80,
		ResponseTime995thPercentileMS: 120,
		SuccessfulRequests:            20,
		ShortCircuitedRequests:        0,
		ThreadTimeouts:                0,
		ThreadPoolRejections:          0,
		Failures:                      0,
		ErrorPercentage:               0.0,
	},
	{
		ServiceName:                   "AuthService",
		CircuitClosed:                 true,
		RequestRatePerHost:            1,
		RequestRatePerCluster:         2,
		HostsInCluster:                2,
		MedianResponseTimeMS:          40,
		MeanResponseTimeMS:            55,
		ResponseTime90thPercentileMS:  60,
		ResponseTime99thPercentileMS:  80,
		ResponseTime995thPercentileMS: 120,
		SuccessfulRequests:            20,
		ShortCircuitedRequests:        0,
		ThreadTimeouts:                0,
		ThreadPoolRejections:          0,
		Failures:                      0,
		ErrorPercentage:               0.0,
	},
	{
		ServiceName:                   "CacheService",
		CircuitClosed:                 false,
		RequestRatePerHost:            1,
		RequestRatePerCluster:         2,
		HostsInCluster:                2,
		MedianResponseTimeMS:          40,
		MeanResponseTimeMS:            55,
		ResponseTime90thPercentileMS:  60,
		ResponseTime99thPercentileMS:  80,
		ResponseTime995thPercentileMS: 120,
		SuccessfulRequests:            20,
		ShortCircuitedRequests:        0,
		ThreadTimeouts:                0,
		ThreadPoolRejections:          0,
		Failures:                      0,
		ErrorPercentage:               0.0,
	},
}

var count int

func (s *Server) hystrixHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := log.FromContext(r.Context())

		// compute some new values
		newValues()

		// figure out the format used to serialize the response
		preferredContentType := httputil.NegotiateContentType(
			r,
			[]string{"text/plain", "application/json", "application/x-protobuf"},
			"application/json",
		)

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Vary", "Accept,Accept-Encoding")

		switch preferredContentType {
		case "text/plain":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)

		case "application/json":
			jw := json.NewEncoder(w)
			/*
				count++
				var response []byte
				if count&1 == 1 {
					response = []byte(response1)
				} else {
					response = []byte(response0)
				}
			*/
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

func newValues() {
	response[0].SuccessfulRequests = rand.Intn(20)
	response[0].Failures = rand.Intn(3)
	response[1].SuccessfulRequests = rand.Intn(10) + 5
	response[1].Failures = rand.Intn(10)
	response[0].ErrorPercentage = computeErrorPercentage(&response[0])
	response[1].ErrorPercentage = computeErrorPercentage(&response[1])

	response[0].CircuitClosed = true
	response[1].CircuitClosed = true
	if response[0].ErrorPercentage > 40.0 {
		response[0].CircuitClosed = false
	}
	if response[1].ErrorPercentage > 40.0 {
		response[1].CircuitClosed = false
	}
}

func computeErrorPercentage(r *HystrixGaugeInfo) float32 {
	sum := r.SuccessfulRequests +
		r.Failures +
		r.ThreadPoolRejections +
		r.ThreadTimeouts +
		r.ShortCircuitedRequests

	if sum == 0 {
		return 0.0
	}

	return roundFloat(float64(sum-r.SuccessfulRequests)/float64(sum)*100.0, 1)
}

func roundFloat(val float64, precision int) float32 {
	ratio := math.Pow(10, float64(precision))
	return float32(math.Round(val*ratio) / ratio)
}
