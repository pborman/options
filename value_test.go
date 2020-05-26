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

package options_test

// Demonstrate that user defined values can be used.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/pborman/options"
	"github.com/pborman/getopt/v2"
)

type myType struct {
	File string
	Line int
}

var (
	errFormat = errors.New("format error, need FILE:LINE")
	errLine   = errors.New("line numbers must be positive integers")
)

func (m *myType) Set(s string, _ getopt.Option) error {
	x := strings.Index(s, ":")
	if x < 0 {
		return errFormat
	}
	n, err := strconv.Atoi(s[x+1:])
	if n < 1 || err != nil {
		return errLine
	}
	m.File = s[:x]
	m.Line = n
	return nil
}

func (m *myType) String() string {
	if *m == (myType{}) {
		return ""
	}
	return fmt.Sprintf("%s:%d", m.File, m.Line)
}

type myOpts struct {
	Location myType `getopt:"--loc=FILE:LINE a location"`
}

func (m *myOpts) String() string {
	if m == nil {
		return "nil"
	}
	if *m == (myOpts{}) {
		return "empty"
	}
	return m.Location.String()
}

func TestUserType(t *testing.T) {

	for _, tt := range []struct {
		name    string
		in      string
		inOpts  *myOpts
		outOpts *myOpts
		err     error
	}{
		{
			name: "no arguments",
		},
		{
			name: "bad format",
			in:   "test --loc=here",
			err:  errFormat,
		},
		{
			name: "bad line",
			in:   "test --loc=here:",
			err:  errLine,
		},
		{
			name: "negative line",
			in:   "test --loc=here:-1",
			err:  errLine,
		},
		{
			name: "pass",
			in:   "test --loc=this_file.go:42",
			outOpts: &myOpts{
				Location: myType{File: "this_file.go", Line: 42},
			},
		},
		{
			name: "default",
			inOpts: &myOpts{
				Location: myType{File: "this_file.go", Line: 42},
			},
			outOpts: &myOpts{
				Location: myType{File: "this_file.go", Line: 42},
			},
		},
		{
			name: "override",
			in:   "test --loc=this_file.go:42",
			inOpts: &myOpts{
				Location: myType{File: "that_file.go", Line: 17},
			},
			outOpts: &myOpts{
				Location: myType{File: "this_file.go", Line: 42},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.inOpts == nil {
				tt.inOpts = &myOpts{}
			}
			if tt.outOpts == nil {
				tt.outOpts = &myOpts{}
			}
			set := getopt.New()
			if err := options.RegisterSet("", tt.inOpts, set); err != nil {
				t.Fatalf("RegisteSet: %v", err)
			}
			err := set.Getopt(strings.Fields(tt.in), nil)
			// Unwrap the error
			if ge, ok := err.(*getopt.Error); ok {
				err = ge.Err
			}
			if err != tt.err {
				t.Fatalf("got error '%v', want '%v'", err, tt.err)
			}
			if *tt.outOpts != *tt.inOpts {
				t.Fatalf("got %v, want %v", tt.inOpts, tt.outOpts)
			}
		})
	}
}
