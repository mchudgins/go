package hystrix

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/afex/hystrix-go/hystrix"
	"go.uber.org/zap"
)

type HTTPClient struct {
	http.Client
	HystrixCommandName string
	logger             *zap.Logger
}

func NewClient(commandName string, logger *zap.Logger) *HTTPClient {
	return &HTTPClient{
		HystrixCommandName: commandName,
		logger:             logger.With(zap.String("hystrixCommand", commandName)),
	}
}

func circuitBreaker(u, commandName string, logger *zap.Logger, fn func() (*http.Response, error)) (*http.Response, error) {

	output := make(chan *http.Response, 1)
	defer close(output)
	errors := make(chan error, 1)
	defer close(errors)

	hystrix.Go(commandName, func() error {
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
		logger.Info("breaker closed", zap.String("url", u), zap.Error(err))
		return err
	})

	select {
	case r := <-output:
		return r, nil

	case err := <-errors:
		return nil, err
	}
}

func (c *HTTPClient) Do(r *http.Request) (*http.Response, error) {
	return circuitBreaker(r.URL.Path, c.HystrixCommandName, c.logger, func() (*http.Response, error) {
		return c.Client.Do(r)
	})
}

func (c *HTTPClient) Get(url string) (*http.Response, error) {
	return circuitBreaker(url, c.HystrixCommandName, c.logger, func() (*http.Response, error) {
		return c.Client.Get(url)
	})
}

func (c *HTTPClient) Head(url string) (*http.Response, error) {
	return circuitBreaker(url, c.HystrixCommandName, c.logger, func() (*http.Response, error) {
		return c.Client.Head(url)
	})

}

func (c *HTTPClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	return circuitBreaker(url, c.HystrixCommandName, c.logger, func() (*http.Response, error) {
		return c.Client.Post(url, contentType, body)
	})
}

func (c *HTTPClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return circuitBreaker(url, c.HystrixCommandName, c.logger, func() (*http.Response, error) {
		return c.Client.PostForm(url, data)
	})
}
