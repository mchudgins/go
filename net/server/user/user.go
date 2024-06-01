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

package user

import (
	"context"
	"fmt"
	"net/http"
)

type key struct{}

const (
	// USERID is an HTTP header
	USERID string = "X-Remote-User"
)

var (
	// NotFound returned when the USERID header is not in the request
	NotFound error = fmt.Errorf("%s not found", USERID)
	userID         = key{}
)

//type key struct{}

// FromRequest gets the userid from an HTTP request
func FromRequest(req *http.Request) (string, error) {
	var err error

	id := req.Header.Get(USERID)
	if len(id) == 0 {
		return "", NotFound
	}

	return id, err
}

// FromContext extracts a user from a Context
func FromContext(ctx context.Context) string {
	val, ok := ctx.Value(userID).(string)
	if ok {
		return val
	}
	return ""
}

// NewContext returns a new Context that carries the provided user id
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userID, id)
}
