/*
 * Copyright (c) 2024.  Mike Hudgins <mchudgins@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package helper

import (
	"runtime/debug"
	"strings"
	"sync"

	"go.uber.org/zap"
)

func TrapPanics(logger *zap.Logger) {
	if r := recover(); r != nil {
		stk := debug.Stack()
		lines := strings.Split(string(stk), "\n")
		logger.Error("panic occurred",
			zap.Any("recoverInfo", r),
			zap.Strings("stack", lines),
		)
	}

}

func LaunchGoRoutine(logger *zap.Logger, wg *sync.WaitGroup, f func()) {

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		// set up the deferred functions
		if wg != nil {
			defer wg.Done()
		}
		defer TrapPanics(logger)

		// now call the actual function to run on the go thread
		f()
	}()
}
