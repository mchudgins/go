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

package simple

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
)

func Run(args []string) {
	var name string

	fmt.Print("\033[H\033[2J") // clear screen

	form := huh.NewForm(
		huh.NewGroup(huh.NewNote().
			Title("Charmburger").
			Description("Welcome to _Charmburger™_.\n\nHow may we take your order?\n\n").
			Next(true).
			NextLabel("Next"),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("What’s your name?").
				Validate(func(str string) error {
					if str == "Frank" {
						return errors.New("Sorry, we don’t serve customers named Frank.")
					}
					return nil
				}).
				Value(&name),
		),
	)
	err := form.Run() //form.WithHeight(40).WithWidth(50).Run() // this is blocking...
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	} else {

		fmt.Printf("Hey, %s!\n", name)
	}
}
