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
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/mchudgins/go/echo/webapp"
	"github.com/mchudgins/go/log"
	"github.com/mchudgins/go/net"
	"github.com/mchudgins/go/net/server"
	"github.com/mchudgins/go/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	listenPort = 9090
	publicKey  = ""
	privateKey = ""
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		options := server.OptionsFactory(
			server.WithHTTPListenPort(listenPort),
			server.WithHTTPServer(api),
			server.WithLogger(webLogger),
			server.WithGzip(),
			server.WithShutdownSignal(stop, wg),
		)
		if len(publicKey) > 0 && len(privateKey) > 0 {
			tlsConfig := net.NewPublicTLSConfig()
			options = append(options,
				server.WithCertificate(publicKey, privateKey),
				server.WithTLSConfig(tlsConfig),
			)
		}

		server.Run(
			options...,
		)

		<-sigs // Wait for signals (this hangs until a signal arrives)
		logger.Info("Shutting down...")

		close(stop) // Tell goroutines to stop themselves
		wg.Wait()   // Wait for all go routines to be stopped
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVar(&listenPort, "listen-port", 9090, "port to listen on")
	serverCmd.Flags().StringVar(&publicKey, "public-key", "", "public key filename")
	serverCmd.Flags().StringVar(&privateKey, "private-key", "", "private key filename")
}
