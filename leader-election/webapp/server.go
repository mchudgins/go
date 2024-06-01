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
	"net/http"

	"github.com/justinas/alice"
	"go.uber.org/zap"

	leader_election "github.com/mchudgins/go/leader-election"
	"github.com/mchudgins/go/services/generic/healthCheck"
)

type WebApp struct {
	healthCheck.UnimplementedHealthServer
	logger         *zap.Logger
	router         *http.ServeMux
	chain          alice.Chain
	LeaderElection *leader_election.LeaderElection
}

func NewServer(logger *zap.Logger) *WebApp {
	s := &WebApp{
		logger:         logger,
		router:         http.NewServeMux(),
		chain:          alice.New(),
		LeaderElection: &leader_election.LeaderElection{},
	}

	s.routes()

	s.chain.Then(s)

	return s
}

func (s *WebApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
