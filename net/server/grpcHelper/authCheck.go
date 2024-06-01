/*
 * Copyright © 2022.  Mike Hudgins <mchudgins@gmail.com>
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

package grpcHelper

import (
	"context"

	"github.com/mchudgins/go/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AuthenticationCheck(approvedClients []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		logger := log.FromContext(ctx)

		logger.Debug("authenticationCheck+")
		defer logger.Debug("authenticationCheck-")

		remoteUser, remoteIP, err := CallerInfo(ctx)
		if err != nil {
			logger.Error("Unauthenticated access attempt", log.SecurityMarker, zap.String("remoteIP", remoteIP))
			return nil, status.Error(codes.Unauthenticated, "Unauthenticated")
		}

		ok := false
		for _, approvedClient := range approvedClients {
			if remoteUser == approvedClient {
				ok = true
				break
			}
		}

		if !ok {
			logger.Error("Unauthorized access by known endpoint",
				log.UnauthorizedMarker,
				zap.String("remoteUser", remoteUser),
				zap.String("remoteIP", remoteIP))
			return nil, status.Error(codes.Unauthenticated, "Unauthenticated")
		}

		return handler(ctx, req)
	}
}
