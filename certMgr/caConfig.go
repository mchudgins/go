/*
 * Copyright © 2025.  Mike Hudgins <mchudgins@gmail.com>
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 */

package certMgr

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func NewCertificateAuthority(caName string,
	certFile string,
	keyFile string,
	bundleFile string) (*ca, error) {
	cert, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("while reading certificate file %s -- %w", certFile, err)
	}

	key, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("while reading key file %s -- %w", keyFile, err)
	}

	bundle, err := os.ReadFile(bundleFile)
	if err != nil {
		return nil, fmt.Errorf("while reading certificate file %s -- %w", certFile, err)
	}

	return createCA(caName, []byte(cert), []byte(key), bundle)
}

/*
func loadAsset(asset string) (string, error) {
	b, err := assets.Asset(asset)
	if err != nil {
		log.WithError(err).WithField("asset", asset)
		return "", err
	}
	return string(b), nil
}

func NewCertificateAuthorityFromConfig(cfg *certMgr.AppConfig) (*ca, error) {
	var err error
	duration := time.Duration(cfg.Backend.MaxDuration)
	_ = duration

	// find the public portion of the Signing CA
	cert := cfg.Backend.SigningCACertificate
	if len(cert) == 0 {
		cert, err = loadAsset("static/signing-ca.crt")
		if err != nil {
			log.WithError(err).Fatal("Application misconfigured, exiting.")
		}
	}

	// find the bundle of intermediate CA's
	bundle := cfg.Backend.Bundle
	if len(bundle) == 0 {
		bundle, err = loadAsset("static/ca-bundle.pem")
		if err != nil {
			log.WithError(err).Fatal("Application misconfigured, exiting.")
		}
	}

	key, err := utils.FindAndReadFile(cfg.Backend.SigningCAKeyFilename, "CA key")
	if err != nil {
		log.WithError(err).Fatalf("Application misconfigured, exiting")
	}

	return createCA("", []byte(cert), []byte(key), bundle)
}

*/

func createCA(caName string,
	cert []byte,
	key []byte,
	bundle []byte) (*ca, error) {

	if len(caName) == 0 {
		caName = "default"
	}

	pemCert, _ := pem.Decode(cert)
	if pemCert == nil {
		msg := "unable to decode the certificate"
		return nil, errors.New(msg)
	}

	pemKey, _ := pem.Decode(key)
	if pemKey == nil {
		msg := "unable to decode the certificate's key"
		return nil, errors.New(msg)
	}

	if x509.IsEncryptedPEMBlock(pemKey) {
		msg := "certificate key requires a passphrase! This is unsupported"
		return nil, errors.New(msg)
	}
	caKey, err := x509.ParsePKCS8PrivateKey(pemKey.Bytes)
	if err != nil {
		msg := "unable to parse certificate's key"
		return nil, errors.New(msg)
	}

	if _, ok := caKey.(crypto.Signer); !ok {
		msg := "hmmm, the CA private key is not a crypto.Signer"
		return nil, errors.New(msg)
	}

	caCertificate, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.New("error parsing CA certificate")

	}

	// log.Infof("permittedDomains:  %s", strings.Join(caCertificate.PermittedDNSDomains, ", "))

	return &ca{Name: caName,
		SigningCertificate: *caCertificate,
		SigningKey:         caKey.(crypto.Signer),
		Bundle:             bundle}, nil
}
