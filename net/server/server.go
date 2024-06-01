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

package server

import (
	"crypto/tls"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	afex "github.com/afex/hystrix-go/hystrix"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	ecconet "github.com/mchudgins/go/net"
	gsh "github.com/mchudgins/go/net/server/handler"
)

// Config holds the set of options used by a server
type Config struct {
	Insecure                bool
	Compress                bool // if true, add compression handling to messages
	UseTracer               bool // if true, add request tracing
	CertFilename            string
	KeyFilename             string
	HTTPListenPort          int
	RPCListenPort           int
	MetricsListenPort       int
	Handler                 http.Handler
	Hostname                string // if present, enforce canonical hostnames
	RPCRegister             RPCRegistration
	logger                  *zap.Logger
	rpcServer               *grpc.Server
	httpServer              *http.Server
	metricsServer           *http.Server
	serviceName             string
	tlsConfig               *tls.Config
	clientAuth              tls.ClientAuthType
	metricsHandler          http.Handler
	shutdown                chan struct{}
	wg                      *sync.WaitGroup
	RPCUnaryInterceptorList []grpc.UnaryServerInterceptor
}

// Option permits changes from the default Config
type Option func(*Config) error

// RPCRegistration is used with WithRPCServer and provides
// the gRPC registration function
type RPCRegistration func(*grpc.Server) error

const (
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"
)

// WithCanonicalHost causes the server to redirect to the specified
// canonical when the request refers to a non-canonical name.
// Useful for public-facing endpoints when trying to perform SEO.
func WithCanonicalHost(hostname string) Option {
	return func(cfg *Config) error {
		cfg.Hostname = hostname

		return nil
	}
}

// WithCertificate provides the x509 public/private keypair.
// also ensures the HTTP/GRPC endpoints use TLS.
func WithCertificate(certFilename, keyFilename string) Option {
	return func(cfg *Config) error {
		cfg.CertFilename = certFilename
		cfg.KeyFilename = keyFilename
		cfg.Insecure = false
		return nil
	}
}

// WithRequestClientCert indicates that the client should send
// a cert, if available.  Only useful if WithCertificate has been set
func WithRequestClientCert() Option {
	return func(cfg *Config) error {
		cfg.clientAuth = tls.VerifyClientCertIfGiven
		return nil
	}
}

// WithHTTPListenPort changes the listen port
func WithHTTPListenPort(port int) Option {
	return func(cfg *Config) error {
		cfg.HTTPListenPort = port
		return nil
	}
}

// WithHTTPServer instructs the server to listen for HTTP/S requests
func WithHTTPServer(h http.Handler) Option {
	return func(cfg *Config) error {
		cfg.Handler = h

		if cfg.httpServer != nil {
			return nil
		}

		cfg.httpServer = &http.Server{
			IdleTimeout:       120 * time.Second,
			ReadTimeout:       500 * time.Millisecond,
			ReadHeaderTimeout: 250 * time.Millisecond,
			WriteTimeout:      2500 * time.Millisecond,
			TLSConfig:         cfg.tlsConfig,
		}

		return nil
	}
}

// WithLogger sets the zap logger
func WithLogger(l *zap.Logger) Option {
	return func(cfg *Config) error {
		cfg.logger = l
		return nil
	}
}

// WithMetricsListenPort changes the listen port for /metrics
func WithMetricsListenPort(port int) Option {
	return func(cfg *Config) error {
		cfg.MetricsListenPort = port
		return nil
	}
}

// WithMetricsServer instructs the server on how to handle readiness/liveness queries
func WithMetricsServer(h http.Handler) Option {
	return func(cfg *Config) error {
		cfg.metricsHandler = h
		return nil
	}
}

// WithRPCListenPort changes the listen port for gRPC
func WithRPCListenPort(port int) Option {
	return func(cfg *Config) error {
		cfg.RPCListenPort = port
		return nil
	}
}

// WithRPCServer instructs the server to listen for gRPC requests
func WithRPCServer(fn RPCRegistration) Option {
	return func(cfg *Config) error {
		cfg.RPCRegister = fn

		return nil
	}
}

// WithRPCUnaryInterceptors adds additional interceptors (beyond logging & metrics)
func WithRPCUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(cfg *Config) error {
		list := []grpc.UnaryServerInterceptor{}
		cfg.RPCUnaryInterceptorList = append(list, interceptors...)

		return nil
	}
}

// WithGzip compresses responses if Accept-Encoding indicates it is desired
func WithGzip() Option {
	return func(cfg *Config) error {
		cfg.Compress = true

		return nil
	}
}

// WithServiceName sets the Tracer service name
func WithServiceName(serviceName string) Option {
	return func(cfg *Config) error {
		cfg.serviceName = serviceName
		return nil
	}
}

// WithTracer enables open tracing of requests
func WithTracer() Option {
	return func(cfg *Config) error {
		cfg.UseTracer = true
		return nil
	}
}

// WithPublicEndpoint informs the server that requests
// are arriving directly from the internet
func WithPublicEndpoint() Option {
	return func(cfg *Config) error {
		cfg.Insecure = false
		cfg.tlsConfig = ecconet.NewPublicTLSConfig()

		cfg.httpServer = &http.Server{
			IdleTimeout:       120 * time.Second,
			ReadTimeout:       250 * time.Millisecond,
			ReadHeaderTimeout: 200 * time.Millisecond,
			WriteTimeout:      250 * time.Millisecond,
			TLSConfig:         cfg.tlsConfig,
		}

		return nil
	}
}

func WithShutdownSignal(c chan struct{}, wg *sync.WaitGroup) Option {
	return func(cfg *Config) error {
		cfg.shutdown = c
		cfg.wg = wg

		return nil
	}
}

// WithTLSConfig allows a specific tls.Config to be used.
// Mutually exclusive with WithPublicEndpoint.
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(cfg *Config) error {
		cfg.Insecure = false
		cfg.tlsConfig = tlsConfig

		return nil
	}
}

// Run starts the configured servers.
func Run(opts ...Option) {

	// default config
	cfg := &Config{
		Insecure:          true,
		HTTPListenPort:    8443,
		MetricsListenPort: 8080,
		RPCListenPort:     50050,
		tlsConfig:         ecconet.NewTLSConfig(),
	}

	// process the Run() options
	for _, o := range opts {
		err := o(cfg)
		if err != nil {
			panic("setting server options -- " + err.Error())
		}
	}

	// make a channel to listen on events,
	// then launch the servers.

	errc := make(chan eventSource)
	var wg *sync.WaitGroup

	// if caller didn't pass a shutdown signal, create a go func to listen for signals
	if cfg.wg == nil {
		wg = &sync.WaitGroup{}
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			errc <- eventSource{
				source: interrupt,
				err:    fmt.Errorf("%s", <-c),
			}
		}()
	} else {
		wg = cfg.wg
		wg.Add(1)
		go func() {
			defer cfg.logger.Debug("signal monitor routine has exited")
			<-cfg.shutdown
			wg.Done()
		}()
	}

	defer close(errc)

	// gRPC server
	if cfg.RPCRegister != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cfg.logger.Debug("rpc go routine has exited")

			rpcListenPort := ":" + strconv.Itoa(cfg.RPCListenPort)
			lis, err := net.Listen("tcp", rpcListenPort)
			if err != nil {
				errc <- eventSource{
					err:    err,
					source: rpcServer,
				}
				return
			}

			// configure the RPC server
			interceptors := []grpc.UnaryServerInterceptor{grpc_prometheus.UnaryServerInterceptor}

			if cfg.logger != nil {
				interceptors = append(interceptors,
					gsh.RPCEndpointLog(cfg.logger, cfg.serviceName))
			}
			/*
				if cfg.UseTracer {
						interceptors = append(interceptors,
							otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer(),
								otgrpc.LogPayloads()))
				}
			*/
			if len(cfg.RPCUnaryInterceptorList) > 0 {
				interceptors = append(interceptors, cfg.RPCUnaryInterceptorList...)
			}
			grpcMiddleware := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(interceptors...))

			if cfg.Insecure {
				cfg.rpcServer = grpc.NewServer(
					grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
					grpcMiddleware)
			} else {
				// load the necessary certificates, etc. to establish a connection
				// secured by mutual authentication over TLS
				cert, err := tls.LoadX509KeyPair(cfg.CertFilename, cfg.KeyFilename)
				if err != nil {
					panic(fmt.Sprintf("unable to load certificate (certificate file %s / key file %s) -- %s\n",
						cfg.CertFilename, cfg.KeyFilename, err))
				}
				tlsConfig := ecconet.NewTLSConfig()
				tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
				tlsConfig.Certificates = []tls.Certificate{cert}

				creds := credentials.NewTLS(tlsConfig)

				cfg.rpcServer = grpc.NewServer(
					grpc.Creds(creds),
					grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
					grpcMiddleware)
			}

			err = cfg.RPCRegister(cfg.rpcServer)
			if err != nil {
				panic(fmt.Sprintf("unable to register RPC endpoint -- %s", err.Error()))
			}

			// register w. prometheus
			grpc_prometheus.Register(cfg.rpcServer)
			grpc_prometheus.EnableHandlingTimeHistogram()

			// run the server
			err = cfg.rpcServer.Serve(lis)
			if err != nil && cfg.logger != nil {
				cfg.logger.Debug("rpcServer has terminated with error",
					zap.Error(err))
			}
		}()
	}

	// http/https server
	if cfg.Handler != nil {
		wg.Add(1)
		go func() {
			var err error
			defer wg.Done()
			defer cfg.logger.Debug("http go routine has exited")

			rootMux := mux.NewRouter()

			rootMux.PathPrefix("/").Handler(cfg.Handler)

			chain := alice.New(gsh.HTTPMetricsCollector, gsh.HTTPAccessLogger(cfg.logger))

			/*
				if cfg.UseTracer {
						var tracer func(http.Handler) http.Handler

						t, err := gsh.NewTracer(cfg.serviceName)
						if err != nil {
							cfg.logger.Panic("unable to construct NewTracer", zap.Error(err))
						}
						tracer = gsh.TracerFromHTTPRequest(t, "http")
						chain.Append(tracer)
				}
			*/

			if len(cfg.Hostname) > 0 {
				canonical := handlers.CanonicalHost(cfg.Hostname, http.StatusPermanentRedirect)
				chain = chain.Append(canonical)
			}

			if cfg.Compress {
				chain = chain.Append(handlers.CompressHandler)
			}

			cfg.httpServer.ConnState = gsh.HTTPConnectionMetricsCollector

			httpListenAddress := ":" + strconv.Itoa(cfg.HTTPListenPort)
			cfg.httpServer.Addr = httpListenAddress
			cfg.httpServer.Handler = chain.Then(rootMux)
			cfg.httpServer.TLSConfig = cfg.tlsConfig

			if cfg.Insecure {
				err = cfg.httpServer.ListenAndServe()
			} else {
				if cfg.clientAuth != tls.NoClientCert {
					cfg.httpServer.TLSConfig.ClientAuth = cfg.clientAuth
				}

				err = cfg.httpServer.ListenAndServeTLS(cfg.CertFilename, cfg.KeyFilename)
			}

			if err == http.ErrServerClosed {
				cfg.logger.Info("http server closed.")
			} else {
				errc <- eventSource{
					err:    err,
					source: httpServer,
				}
			}
		}()
	}

	// start the metrics/hystrix/health stream provider
	if cfg.metricsHandler != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cfg.logger.Debug("metrics go routine has exited")

			rootMux := http.NewServeMux()

			chain := alice.New(gsh.HTTPMetricsCollector, gsh.HTTPAccessLogger(cfg.logger))

			hystrixStreamHandler := afex.NewStreamHandler()
			hystrixStreamHandler.Start()

			rootMux.Handle("/debug/vars", expvar.Handler())
			rootMux.Handle("/hystrix", hystrixStreamHandler)
			rootMux.Handle("/metrics", promhttp.Handler())
			rootMux.Handle("/", cfg.metricsHandler)

			listenPort := ":" + strconv.Itoa(cfg.MetricsListenPort)
			cfg.metricsServer = &http.Server{
				Addr:      listenPort,
				Handler:   chain.Then(rootMux),
				ConnState: gsh.HTTPConnectionMetricsCollector,
			}

			err := cfg.metricsServer.ListenAndServe()
			if err == http.ErrServerClosed {
				err = nil
			}
			errc <- eventSource{
				err:    err,
				source: metricsServer,
			}
		}()
	}

	cfg.logLaunch()

	if cfg.wg != nil {
		cfg.wg.Add(1)
		go func() {
			defer cfg.wg.Done()
			defer cfg.logger.Debug("shutdown monitor go routine has exited")

			// wait for somthin'
			<-cfg.shutdown

			// somethin happened, now shut everything down gracefully, if possible
			rc := eventSource{
				source: unknown,
				err:    nil,
			}
			cfg.logger.Debug("shutdown channel closed. Initiating Graceful Shutdown")
			cfg.performGracefulShutdown(errc, rc)
		}()

		return
	}

	// wait for somthin'
	rc := <-errc
	cfg.logger.Debug("somthin happend")
	// somethin happened, now shut everything down gracefully, if possible
	cfg.performGracefulShutdown(errc, rc)
	// close(errc)
}

func (cfg *Config) logLaunch() {
	if cfg.logger == nil {
		return
	}

	serverList := make([]zapcore.Field, 0, 3)

	if cfg.RPCRegister != nil {
		serverList = append(serverList, zap.Int("gRPC_port", cfg.RPCListenPort))
	}
	if cfg.Handler != nil {
		var key = "HTTPS_port"
		if cfg.Insecure {
			key = "HTTP_port"
		}
		serverList = append(serverList, zap.Int(key, cfg.HTTPListenPort))
	}
	if cfg.metricsHandler != nil {
		serverList = append(serverList, zap.Int("metrics_port", cfg.MetricsListenPort))
	}

	if cfg.Insecure {
		cfg.logger.Info("Server listening insecurely on one or more ports", serverList...)
	} else {
		cfg.logger.Info("Server", serverList...)
	}
}

// OptionsFactory is a convenience function to build a slice of Options for the variadic Run() method
// Run() can be used directly without OptionsFactory, but sometimes it is desirable
// to manipulate the list of Options at runtime.
func OptionsFactory(opts ...Option) []Option {
	options := make([]Option, 0)
	options = append(options, opts...)

	return options
}
