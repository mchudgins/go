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
	"fmt"
	"github.com/go-logr/zapr"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/mchudgins/go/helper"
	"github.com/mchudgins/go/leader-election/webapp"
	"github.com/mchudgins/go/net/server"
	"github.com/mchudgins/go/net/server/grpcHelper"
	"github.com/mchudgins/go/version"
)

const (
	lockName       = "k8s-leader-example"
	leaseNamespace = "default"
)

var (
	// ENV options
	leaseName = "k8s-leader-example"
)

func Run(logger *zap.Logger, httpPort int) {
	namespace := os.Getenv("POD_NAMESPACE")
	if len(os.Getenv("LEASE_NAME")) > 0 {
		leaseName = os.Getenv("LEASE_NAME")
	}

	podName := os.Getenv("POD_NAME")

	logger.Info("starting up",
		zap.String("configFilename", viper.ConfigFileUsed()),
		zap.String("version", version.VERSION),
		zap.String("gitCommit", version.GITCOMMIT),
		zap.String("POD_NAMESPACE", namespace),
		zap.String("POD_NAME", podName),
		zap.String("NODE_NAME", os.Getenv("NODE_NAME")),
		zap.String("defaultNamespace", metav1.NamespaceDefault),
		zap.String("HOME", os.Getenv("HOME")))

	// set up OS signals & waitgroups
	sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
	stop := make(chan struct{})     // Create channel to receive stop signa
	wg := &sync.WaitGroup{}

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receive SIGTERM

	for {
		select {
		case <-time.After(5 * time.Second):
			logger.Info("still alive")

		case <-sigs:
			logger.Info("signal received")
			os.Exit(0)

		case <-stop:
			logger.Info("stop received")
			os.Exit(0)
		}
	}

	klog.SetLogger(zapr.NewLogger(logger)) // have the client-go library use the zap logger

	// Create a Kubernetes client using the current context
	// recover from any panic from the OrDie attempts within
	var clientset *kubernetes.Clientset
	clientset = func(logger *zap.Logger) *kubernetes.Clientset {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("unable to obtain kubernetes client set")
				clientset = nil
				os.Exit(1)
			}
		}()

		clientset = kubernetes.NewForConfigOrDie(cruntimeconfig.GetConfigOrDie())

		return clientset
	}(logger)
	if clientset == nil {
		logger.Error("unable to obtain k8s client")
		os.Exit(0)
	}

	logger.Info("k8s clientset obtained")

	stopLease, err := MonitorLease(logger, wg, clientset, namespace, leaseName, podName)
	if err != nil {
		logger.Fatal("unable to monitor lease",
			zap.Error(err))
	}

	helper.LaunchGoRoutine(logger.With(zap.String("goRoutine", "monitorPods")), wg, func() {
		for {
			select {
			case <-time.After(60 * time.Second):
				pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					logger.Warn("unable to return list of pods",
						zap.Error(err))
				} else {
					for _, v := range pods.Items {
						logger.Info("pod", zap.String("name", v.Name))
					}
				}

			case <-stop:
				logger.Info("stop signal received")
				return
			}
		}
	})

	// start up the http & grpc servers

	weblogger := logger.With(zap.String("mod", "webapp"))
	s := webapp.NewServer(weblogger)
	options := server.OptionsFactory(
		server.WithHTTPServer(s),
		server.WithRPCUnaryInterceptors(grpcHelper.Recovery),
		server.WithRPCServer(func(g *grpc.Server) error {
			h := health.NewServer()
			healthgrpc.RegisterHealthServer(g, h)

			return nil
		}),
		server.WithShutdownSignal(stop, wg),
		server.WithHTTPListenPort(httpPort),
		server.WithServiceName("leaderElection"),
		server.WithLogger(weblogger),
		server.WithGzip(),
	)

	// start the metrics, liveness, readiness server
	server.Run(options...)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	logger.Info("OS Signal received. Shutting down...")

	stopLease() // Tell the MonitorLease to stop
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all go routines to be stopped
}
