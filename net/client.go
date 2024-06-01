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
//

package net

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http2"
)

// NewClient provides an http.Client suitable for use within the datacenter
func NewClient() *http.Client {
	transport := NewRoundTripper()

	client := http.Client{
		// everything is o' so close!
		Timeout: 5 * time.Second,

		// never follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},

		Transport: transport,
	}

	return &client
}

// NewRoundTripper provides an http.RoundTripper for use within the datacenter
func NewRoundTripper() http.RoundTripper {
	transport := &http.Transport{
		Proxy:                  func(*http.Request) (*url.URL, error) { return nil, nil }, // never explicitly proxy, use transparent proxy
		MaxConnsPerHost:        250,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    100,
		IdleConnTimeout:        0, // never timeout, let the server close
		ResponseHeaderTimeout:  1 * time.Second,
		ExpectContinueTimeout:  100 * time.Millisecond,
		MaxResponseHeaderBytes: 8 * 1024,
		TLSHandshakeTimeout:    250 * time.Millisecond,
		TLSClientConfig:        NewTLSConfig(),
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 5 * time.Minute,
			DualStack: true,
		}).DialContext,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		panic(err)
	}

	return transport
}

// NewInsecureRoundTripper provides an insecure http.RoundTripper for use within the datacenter
func NewInsecureRoundTripper() http.RoundTripper {
	transport := &http.Transport{
		Proxy:                  func(*http.Request) (*url.URL, error) { return nil, nil }, // never explicitly proxy, use transparent proxy
		MaxConnsPerHost:        250,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    100,
		IdleConnTimeout:        0, // never timeout, let the server close
		ResponseHeaderTimeout:  5 * time.Second,
		ExpectContinueTimeout:  100 * time.Millisecond,
		MaxResponseHeaderBytes: 8 * 1024,
		TLSHandshakeTimeout:    250 * time.Millisecond,
		//TLSClientConfig:        NewTLSConfig(),
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 5 * time.Minute,
			DualStack: true,
		}).DialContext,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		panic(err)
	}

	transport.TLSClientConfig.InsecureSkipVerify = true

	return transport
}

// NewRemoteClient provides an http.Client suitable for use
// when contacting an endpoint outside the datacenter
func NewRemoteClient() *http.Client {
	transport := NewRemoteRoundTripper()

	client := http.Client{
		// everything is o' so far away!
		Timeout: 10 * time.Second,

		// never follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},

		Transport: transport,
	}

	return &client
}

// NewRemoteRoundTripper provides an http.RoundTripper suitable for use
// when contacting an endpoint outside the datacenter

func NewRemoteRoundTripper() http.RoundTripper {
	transport := &http.Transport{
		Proxy:                  func(*http.Request) (*url.URL, error) { return nil, nil }, // never explicitly proxy, use transparent proxy
		MaxConnsPerHost:        250,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    100,
		IdleConnTimeout:        0, // never timeout, let the server close
		ResponseHeaderTimeout:  10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		MaxResponseHeaderBytes: 8 * 1024,
		TLSHandshakeTimeout:    5 * time.Second,
		TLSClientConfig:        NewTLSConfig(),
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 5 * time.Minute,
			DualStack: true,
		}).DialContext,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		panic(err)
	}

	return transport
}
