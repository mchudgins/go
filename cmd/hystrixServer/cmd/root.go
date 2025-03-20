/*
Copyright © 2025 Mike Hudgins <mchudgins@gmail.com>

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
	"github.com/mchudgins/go/hystrixServer/webapp"
	"github.com/mchudgins/go/log"
	"github.com/mchudgins/go/net/server"
	"github.com/mchudgins/go/version"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	logLevel   = "INFO"
	asJSON     = false
	listenPort = 9090
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hystrixServer",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		cmdName := ""
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}
		if strings.ToLower(logLevel) == "debug" {
			cmdName = path.Base(exe)
		}
		logger := log.GetCmdLogger(cmdName, logLevel, asJSON)
		logger.Info("starting up",
			zap.String("version", version.VERSION),
			zap.String("gitCommit", version.GITCOMMIT))
		logger.Info("start-up options",
			zap.String("log-level", logLevel),
		)

		// set up OS signals & waitgroups
		sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
		stop := make(chan struct{})     // Create channel to receive stop signal

		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receive SIGTERM

		wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on so that they finish

		// start a server to listen for http messages
		webLogger := logger.With(zap.String("server", "webapp"))
		api := webapp.NewServer(webLogger)
		server.Run(
			server.WithHTTPListenPort(listenPort),
			server.WithHTTPServer(api),
			server.WithLogger(webLogger),
			server.WithGzip(),
			server.WithShutdownSignal(stop, wg),
		)

		<-sigs // Wait for signals (this hangs until a signal arrives)
		logger.Info("Shutting down...")

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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.echo-server.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level {'debug', 'info', 'warn', 'error'}")
}
