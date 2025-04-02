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
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

type ca struct {
	Name               string
	SigningCertificate x509.Certificate
	SigningKey         crypto.Signer
	RootCertificate    x509.Certificate
	Bundle             []byte
}

func (c ca) CreateCertificate(ctx context.Context,
	commonName string,
	alternateNames []string,
	duration time.Duration) (cert string, key string, err error) {
	requestedHosts := make([]string, 1, len(alternateNames)+1)
	requestedHosts[0] = commonName
	copy(requestedHosts[1:], alternateNames)

	hosts, err := c.validateRequest(alternateNames, duration)
	if err != nil {
		return "", "", err
	}

	// create the CSR

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	/*
		keyOut, err := os.Create("key.pem")
		defer keyOut.Close()
		if err != nil {
			log.Fatalf("failed to open key.pem for writing: %s", err)
		}
		pem.Encode(keyOut, pemBlockForKey(priv))
	*/

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	notBefore := time.Now()
	notAfter := notBefore.Add(duration)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   hosts[0],
			Organization: []string{"Hudgins' Prototype"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// sign the CSR
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &c.SigningCertificate, publicKey(priv), c.SigningKey)
	if err != nil {
		return "", "", fmt.Errorf("unable to create certificate: %w", err)
	}

	// prepare the response
	var certBuffer, keyBuffer bytes.Buffer

	/*
		certOut, err := os.Create("cert.pem")
		defer certOut.Close()
		if err != nil {
			log.WithError(err).Fatal("failed to open cert.pem for writing")
		}

		pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	*/

	pem.Encode(&certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	cert = certBuffer.String()
	pem.Encode(&keyBuffer, pemBlockForKey(priv))
	key = keyBuffer.String()

	//	// persist the certificate
	//	serverCert, err := x509.ParseCertificate(derBytes)
	//	persistedCert := newCertFromCertificate(serverCert)
	//
	//	/* using hystrix/circuitbreaker to persist the data */
	//	err = hystrix.Do("certs-mysql", func() error {
	//		persistedCert.Insert()
	//		return nil
	//	}, nil)

	/*
		data := &CertificateData{*serverCert}
		data.Persist(ctx)
	*/

	return cert, key, err
}

// from golang.org/pkg/crypto/x509/verify.go
func matchNameConstraint(domain, constraint string) bool {
	// The meaning of zero length constraints is not specified, but this
	// code follows NSS and accepts them as valid for everything.
	if len(constraint) == 0 {
		return true
	}

	if len(domain) < len(constraint) {
		return false
	}

	prefixLen := len(domain) - len(constraint)
	if !strings.EqualFold(domain[prefixLen:], constraint) {
		return false
	}

	if prefixLen == 0 {
		return true
	}

	isSubdomain := domain[prefixLen-1] == '.'
	constraintHasLeadingDot := constraint[0] == '.'
	return isSubdomain != constraintHasLeadingDot
}

func (c *ca) validateRequest(requestedHosts []string, validFor time.Duration) ([]string, error) {
	var hosts = make([]string, len(requestedHosts))

	for i, s := range requestedHosts {
		supportedDomain := false
		fIPAddr := false

		if ip := net.ParseIP(s); ip != nil {
			fIPAddr = true
			if i == 0 {
				return nil, errors.New("Subject name of Certificate must NOT be an IP address")
			}
			hosts[i] = s
		}

		if !fIPAddr {
			h := strings.ToLower(s)
			hosts[i] = h

			if strings.HasPrefix(h, "www.") {
				return nil, errors.New("www. host names are not supported")
			}

			if strings.HasPrefix(h, ".") {
				return nil, errors.New(". host names are not supported")
			}

			if len(c.SigningCertificate.PermittedDNSDomains) > 0 {
				for _, constraint := range c.SigningCertificate.PermittedDNSDomains {
					if matchNameConstraint(h, constraint) {
						supportedDomain = true
					}
				}
			} else {
				supportedDomain = true
			}

			if !supportedDomain {
				return nil, fmt.Errorf("%s is not a permitted domain", h)
			}

		}
	}

	return hosts, nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			// log.WithError(err).Fatal("Unable to marshall ECDSA private key")
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func (c ca) String() string {
	return "Name: " + c.Name + "; ...."
}
