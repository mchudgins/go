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
	"context"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mchudgins/go/certMgr"
	"github.com/mchudgins/go/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type newCmdConfig struct {
	Config       string `json:"config"`
	CertFilename string `json:"certFilename"`
	KeyFilename  string `json:"keyFilename"`
	Duration     int    `json:"duration"`
	Verbose      bool   `json:"verbose"`

	SigningCertFilename   string `json:"signingCertFilename"`
	SigningKeyFilename    string `json:"signingKeyFilenae"`
	SigningBundleFilename string `json:"signingBundleFilename"`
}

// defaultConfig holds default values
var defaultConfig = &newCmdConfig{
	Config:       "",
	CertFilename: "cert.pem",
	KeyFilename:  "key.pem",
	Duration:     90,

	SigningCertFilename:   "ca/svc/svc-ca.crt",
	SigningKeyFilename:    "ca/svc/private/svc-ca.key",
	SigningBundleFilename: "ca/svc/ca-bundle.pem",
}

// newCmd represents the new command
var newCmd = &cobra.Command{
	Args:  cobra.MinimumNArgs(1),
	Use:   "new <common name> [<subject alternate name>...]",
	Short: "Create a new certificate",
	Long:  `Creates a new certificate and key for the specified common name.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmdName := ""
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}
		if strings.ToLower(logLevel) == "debug" {
			cmdName = path.Base(exe)
		}
		logger := log.GetCmdLogger(cmdName, logLevel, false)

		cfg := defaultConfig

		/*
			// flags need special handling (sigh)
			cfg.CertFilename = viper.GetString("cert")
			cfg.KeyFilename = viper.GetString("key")
			cfg.Duration = viper.GetInt("duration")
			cfg.Verbose = viper.GetBool("verbose")

			cfg.SigningCertFilename = viper.GetString("signerCert")
			cfg.SigningKeyFilename = viper.GetString("signerKey")
			cfg.SigningBundleFilename = viper.GetString("signerBundle")
		*/

		// initialize the SimpleCA
		ca, err := certMgr.NewCertificateAuthority("signingCert",
			cfg.SigningCertFilename,
			cfg.SigningKeyFilename,
			cfg.SigningBundleFilename)
		if err != nil {
			logger.Fatal("unable to initialize the CA", zap.Error(err))
		}

		ctx := context.Background()

		cert, key, err := ca.CreateCertificate(ctx, args[0], args, time.Duration(cfg.Duration)*time.Hour*24)
		if err != nil {
			logger.Fatal("unable to generate & sign certificate",
				zap.Error(err), zap.String("subject name", args[0]))
			os.Exit(1)
		}

		certFile, err := os.OpenFile(cfg.CertFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			logger.Fatal("unable to open file",
				zap.Error(err), zap.String("file", cfg.CertFilename))
			os.Exit(1)
		}
		defer certFile.Close()
		certFile.WriteString(cert)

		bundle, err := os.ReadFile(cfg.SigningBundleFilename)
		certFile.Write(bundle)

		keyFile, err := os.OpenFile(cfg.KeyFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			logger.Fatal("unable to open file",
				zap.Error(err), zap.String("file", cfg.KeyFilename))
			os.Exit(1)
		}
		defer keyFile.Close()
		keyFile.WriteString(key)

	},
}

func init() {
	rootCmd.AddCommand(newCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// newCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// newCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	newCmd.Flags().String("cert", defaultConfig.CertFilename, "output file for the PEM encoded certificate")
	newCmd.Flags().Int("duration", defaultConfig.Duration, "# of days duration for the certificate's validity")
	newCmd.Flags().String("key", defaultConfig.KeyFilename, "output file for the PEM encoded key")
	newCmd.Flags().String("signerCert", defaultConfig.SigningCertFilename, "signer CA certificate file")
	newCmd.Flags().String("signerKey", defaultConfig.SigningKeyFilename, "signer CA key file")
	newCmd.Flags().String("signerBundle", defaultConfig.SigningBundleFilename, "signer CA bundle file")

}
