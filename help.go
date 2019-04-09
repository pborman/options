// Copyright 2019 Paul Borman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and

package options

import (
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
)

// A Help option causes PrintUsage to be called if the the option is set.
// Normally os.Exit(0) will be called when the option is seen.  Setting the
// defaulted value to true will prevent os.Exit from being called.
//
// Normal Usage
//
//	var myOptions = struct {
//		Help options.Help `getopt:"--help display command usage"`
//		...
//	}{}
type Help bool

// Set implements getopt.Value.
func (h *Help) Set(value string, opt getopt.Option) error {
	if !opt.Seen() {
		return nil
	}
	getopt.PrintUsage(os.Stderr)
	if !*h {
		os.Exit(0)
	}
	return nil
}

// String implements getopt.Value.
func (h *Help) String() string {
	return fmt.Sprint(bool(*h))
}
