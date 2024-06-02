/*
Copyright © 2024 Mike Hudgins <mchudgins@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	cruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/mchudgins/go/leader-election"
	lew "github.com/mchudgins/go/leader-election/webapp"
	"github.com/mchudgins/go/log"
	"github.com/mchudgins/go/net/server"
	"github.com/mchudgins/go/net/server/grpcHelper"
	"github.com/mchudgins/go/version"
)

const (
	lockName       = "k8s-leader-example"
	leaseNamespace = "default"
)

var (
	// cli options
	asJSON   bool
	cfgFile  string
	fVerbose bool
	httpPort = 8080
	logLevel string

	// ENV options
	leaseName = "k8s-leader-example"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "leader-election",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

	Run: func(cmd *cobra.Command, args []string) {
		runTime := time.Now()
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}

		namespace := os.Getenv("POD_NAMESPACE")
		if len(os.Getenv("LEASE_NAME")) > 0 {
			leaseName = os.Getenv("LEASE_NAME")
		}

		podName := os.Getenv("POD_NAME")

		logger := log.GetCmdLogger(path.Base(exe), logLevel, asJSON)
		logger.Info("starting up",
			zap.String("configFilename", viper.ConfigFileUsed()),
			zap.String("version", version.VERSION),
			zap.String("gitCommit", version.GITCOMMIT),
			zap.String("POD_NAMESPACE", namespace),
			zap.String("POD_NAME", podName),
			zap.String("NODE_NAME", os.Getenv("NODE_NAME")),
			zap.String("defaultNamespace", metav1.NamespaceDefault))

		rnd := rand.New(rand.NewSource(runTime.UnixNano()))
		_ = rnd

		// Create a Kubernetes client using the current context
		// recover from any panic from the OrDie attempts within
		var clientset *kubernetes.Clientset
		clientset = func(logger *zap.Logger) *kubernetes.Clientset {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("unable to obtain kubernetes client set")
					clientset = nil
				}
			}()

			clientset = kubernetes.NewForConfigOrDie(cruntimeconfig.GetConfigOrDie())

			return clientset
		}(logger)
		if clientset == nil {
			os.Exit(0)
		}
		klog.SetLogger(zapr.NewLogger(logger)) // have the client-go library use the zap logger

		// set up OS signals & waitgroups
		sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
		stop := make(chan struct{})     // Create channel to receive stop signa

		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receive SIGTERM

		wg, err := leader_election.MonitorLease(logger, clientset, namespace, leaseName, podName)
		if err != nil {
			logger.Fatal("unable to monitor lease",
				zap.Error(err))
		}

		go func() {
			for {
				pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					logger.Warn("unable to return list of pods",
						zap.Error(err))
				} else {
					for _, v := range pods.Items {
						logger.Info("pod", zap.String("name", v.Name))
					}
				}

				time.Sleep(60 * time.Second)
			}
		}()

		// start up the http & grpc servers

		weblogger := logger.With(zap.String("mod", "webapp"))
		s := lew.NewServer(weblogger)
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

		close(stop) // Tell goroutines to stop themselves
		wg.Wait()   // Wait for all go routines to be stopped
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mc-parser.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level {'debug', 'info', 'warn', 'error'}")
	rootCmd.PersistentFlags().BoolVarP(&fVerbose, "verbose", "v", false, "log additional details")
	rootCmd.PersistentFlags().BoolVar(&asJSON, "json", false, "use JSON as log output format")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home := homedir.HomeDir()
		exename := filepath.Base(os.Args[0])
		// Search config in home directory with name ".getFleetStatus" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("." + exename)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
	}
}
