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
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/mchudgins/go/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"

	leader_election "github.com/mchudgins/go/leader-election"
)

var (
	// cli options
	asJSON   bool
	cfgFile  string
	fVerbose bool
	httpPort = 8080
	logLevel string
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
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}
		_ = exe

		logger := log.GetCmdLogger( /*path.Base(exe)*/ "", logLevel, asJSON)

		runTime := time.Now()
		rnd := rand.New(rand.NewSource(runTime.UnixNano()))
		_ = rnd

		leader_election.Run(logger, httpPort)
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
