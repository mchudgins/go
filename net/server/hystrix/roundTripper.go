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

package hystrix

import (
	"fmt"
	"net/http"

	"github.com/afex/hystrix-go/hystrix"
	"go.uber.org/zap"
)

type Transport struct {
	transport          http.RoundTripper
	logger             *zap.Logger
	hystrixCommandName string
}

// NewTransport creates a hystrix-wrapped transport
func NewTransport(rt http.RoundTripper, commandName string, logger *zap.Logger) http.RoundTripper {
	t := &Transport{
		transport:          rt,
		logger:             logger.With(zap.String("commandName", commandName)),
		hystrixCommandName: commandName,
	}

	return t
}

func (t *Transport) circuitBreak(req *http.Request, fn func() (*http.Response, error)) (*http.Response, error) {

	output := make(chan *http.Response, 1)
	defer close(output)
	errors := make(chan error, 1)
	defer close(errors)

	hystrix.Go(t.hystrixCommandName, func() error {
		response, err := fn()
		if err != nil {
			errors <- err
		} else {
			output <- response
			if response.StatusCode == http.StatusInternalServerError {
				return fmt.Errorf("error %d", response.StatusCode)
			}
		}

		return err
	}, func(err error) error {
		t.logger.Info("breaker closed",
			zap.String("url",
				req.URL.String()),
			zap.Error(err))
		return err
	})

	select {
	case r := <-output:
		return r, nil

	case err := <-errors:
		return nil, err
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.circuitBreak(req, func() (*http.Response, error) {
		return t.transport.RoundTrip(req)
	})
}
