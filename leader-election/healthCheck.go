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

package leader_election

import (
	"context"
	"github.com/mchudgins/go/net/server/healthcheck"
	"net/http"

	"google.golang.org/grpc"

	"github.com/mchudgins/go/services/generic/healthCheck"
)

func (le *LeaderElection) Check(ctx context.Context, in *healthCheck.HealthCheckRequest, opts ...grpc.CallOption) (*healthCheck.HealthCheckResponse, error) {

	health := &healthCheck.HealthCheckResponse{}

	return health, nil
}

func HealthCheckAPI() http.Handler {
	h := healthcheck.NewHandler()

	h.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(25))

	return h
}
