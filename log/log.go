// Copyright © 2017 Mike Hudgins <mchudgins@gmail.com>
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

/*
log provides the logger utilities & interfaces
*/

package log

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LogLevel = "LOG_LEVEL"
)

var (
	SecurityMarker     = NewMarker("security")
	UnauthorizedMarker = NewMarker("security", "unauthorized")
)

// GetLogger returns a zap.Logger for serverless processes
func GetLambdaLogger(lambdaName string) *zap.Logger {
	// See the documentation for Config and zapcore.EncoderConfig for all the
	// available options.
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase",
		"timeEncoder": "iso8601",
		"durationEncoder": "string"
	  }
	}`)

	config := &zap.Config{}
	if err := json.Unmarshal(rawJSON, config); err != nil {
		panic(err)
	}
	config.InitialFields = make(map[string]interface{}, 1)
	config.InitialFields["lambda"] = lambdaName

	config = setLogLevelFromEnv(config)

	//	config := log.NewDevelopmentConfig()
	//	config.EncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger //.With(log.String("x-request-id", "01234"))
}

// GetCmdLogger returns a zap.Logger suitable for non-lambda processes
func GetCmdLogger(cmdName, logLevel string, asJSON bool) *zap.Logger {
	// See the documentation for Config and zapcore.EncoderConfig for all the
	// available options.
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "development" : false,
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase",
		"timeEncoder": "iso8601",
		"durationEncoder": "string"
	  }
	}`)

	config := &zap.Config{}
	if err := json.Unmarshal(rawJSON, config); err != nil {
		panic(err)
	}
	if asJSON {
		//config.EncoderConfig = zap.NewDevelopmentEncoderConfig()
		config.EncoderConfig = zap.NewProductionEncoderConfig()
		config.EncoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder
		config.EncoderConfig.TimeKey = "ms"
	} else {
		config.Encoding = "console"
		config.EncoderConfig = zap.NewProductionEncoderConfig()
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339Nano)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	config.DisableStacktrace = true
	if len(cmdName) > 0 {
		config.InitialFields = make(map[string]interface{}, 1)
		config.InitialFields["cmd"] = cmdName
	}

	config = SetLogLevel(config, logLevel)

	//	config := log.NewDevelopmentConfig()
	//	config.EncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	// add metrics
	logger = logger.WithOptions(zap.Hooks(PrometheusMetrics))

	return logger //.With(log.String("x-request-id", "01234"))
}

func SetLogLevel(config *zap.Config, level string) *zap.Config {
	switch strings.ToUpper(level) {
	case "DEBUG":
	case "TRACE":
		config.Level.SetLevel(zapcore.DebugLevel)

	case "INFO":
		config.Level.SetLevel(zapcore.InfoLevel)

	case "WARN":
		config.Level.SetLevel(zapcore.WarnLevel)

	default:
		fmt.Printf("Unknown LOG_LEVEL value %s.  Log Level set to INFO.", level)
	}

	return config
}

func setLogLevelFromEnv(config *zap.Config) *zap.Config {
	level := strings.ToUpper(os.Getenv(LogLevel))
	if len(level) == 0 {
		level = "INFO"
	}

	return SetLogLevel(config, level)
}

// NewMarker is a helper function to create a new Marker.
// Marker implements the ability to add categorization to logging events.  They should form a
// hierarchy.
// SecurityMarker := NewMarker("security")
// UnauthorizedMarker := NewMarker("security", "unauthorized")
// logger.Debug("message", SecurityMarker, zap.String("someOtherData", "abc"))
func NewMarker(markers ...string) zapcore.Field {
	return zap.Strings("markers", markers)
}

// TODO add standard markers (in log/markers package ??)
