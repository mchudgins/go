package hystrix

import (
	"fmt"
	"net/http"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/mchudgins/go/net/server/httpWriter"
	"go.uber.org/zap"
)

type hystrixHelper struct {
	commandName string
	logger      *zap.Logger
}

func NewHystrixHelper(commandName string, logger *zap.Logger) (*hystrixHelper, error) {
	hystrix.ConfigureCommand(commandName, hystrix.CommandConfig{
		MaxConcurrentRequests: 100,
	})

	return &hystrixHelper{commandName: commandName,
		logger: logger.With(zap.String("hystrixCommand", commandName))}, nil
}

func (y *hystrixHelper) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := hystrix.Do(y.commandName, func() (err error) {

			monitor := httpWriter.NewHTTPWriter(w)

			h.ServeHTTP(monitor, r)

			rc := monitor.StatusCode()
			if rc >= 500 && rc < 600 {
				//log.Printf("StatusCode indicates backend failure")
				return fmt.Errorf("failure contacting %s", y.commandName)
			}
			return nil
		}, func(err error) error {
			y.logger.Warn("breaker open",
				zap.Error(err))
			return nil
		})
		if err != nil {
			y.logger.Warn("Hystrix Error",
				zap.Error(err))
		}
	})
}
