// Copyright 2018 Paul Borman
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
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pborman/getopt/v2"
	"github.com/pborman/check"
)

type theOptions struct {
	Name    string        `getopt:"--name=NAME      name of the widget"`
	Count   int           `getopt:"--count -c=COUNT number of widgets"`
	Verbose bool          `getopt:"-v               be verbose"`
	N       int           `getopt:"-n=NUMBER        set n to NUMBER"`
	Timeout time.Duration `getopt:"--timeout        duration of run"`
	Lazy    string
	Unused  int `getopt:"-"`
}

var myOptions = theOptions{
	Count: 42,
}

// This is the help we expect from theOptions.  If you change theOptions then
// you must change this string.  Note that getopt.HelpColumn must be set to 25.
var theHelp = `
Usage: program [-v] [-c COUNT] [--lazy value] [-n NUMBER] [--name NAME] [--timeout value] [parameters ...]
 -c, --count=COUNT    number of widgets [42]
     --lazy=value     unspecified
 -n NUMBER            set n to NUMBER
     --name=NAME      name of the widget
     --timeout=value  duration of run
 -v                   be verbose
`[1:]

func TestLookup(t *testing.T) {
	opt := &struct {
		Ignore bool   `getopt:"-"`
		Option string `getopt:"--option -o"`
		Lazy   string
	}{
		Option: "value",
		Lazy:   "lazy",
	}
	if o := Lookup(opt, "option"); o.(string) != "value" {
		t.Errorf("--option returned value %q, want %q", o, "value")
	}
	if o := Lookup(opt, "o"); o.(string) != "value" {
		t.Errorf("-o returned value %q, want %q", o, "value")
	}
	if o := Lookup(opt, "lazy"); o.(string) != "lazy" {
		t.Errorf("--lazy returned value %q, want %q", o, "lazy")
	}
	if o := Lookup("a", "a"); o != nil {
		t.Errorf("string returned %v, want nil", o)
	}
	if o := Lookup(new(string), "a"); o != nil {
		t.Errorf("*string returned %v, want nil", o)
	}
	if o := Lookup(opt, "missgin"); o != nil {
		t.Errorf("missing returned %v, want nil", o)
	}
	opt2 := &struct {
		Invalid string `getopt:"invalid tag"`
		Option  string `getopt:"--option -o"`
		Lazy    string
	}{
		Option: "value",
	}
	if o := Lookup(opt2, "option"); o != nil {
		t.Errorf("able to lookup past invalid tag")
	}
}

func TestValidate(t *testing.T) {
	opts := &struct {
		Name string `getopt:"--the_name"`
	}{}
	if err := Validate(opts); err != nil {
		t.Errorf("Validate returned error %v for valid set", err)
	}
	opts2 := struct {
		Name string `getopt:"the_name"`
	}{}
	if err := Validate(opts2); err == nil {
		t.Errorf("Validate did not return an error for an valid set")
	}
}

func TestHelp(t *testing.T) {
	getopt.HelpColumn = 25
	opts, s := RegisterNew("", &myOptions)
	if dopts, ok := opts.(*theOptions); !ok {
		t.Errorf("RegisterNew returned type %T, want %T", dopts, opts)
	}
	var buf bytes.Buffer
	s.SetProgram("program")
	s.PrintUsage(&buf)
	if help := buf.String(); help != theHelp {
		t.Errorf("Got help:\n%s\nWant:\n%s", help, theHelp)
	}
}

func TestRegisterSet(t *testing.T) {
	opts := &struct {
		Name string `getopt:"--the_name"`
	}{
		Name: "bob",
	}
	s := getopt.New()
	RegisterSet("", opts, s)
	s.VisitAll(func(o getopt.Option) {
		if o.Name() != "--the_name" {
			t.Errorf("unexpected option found: %q", o.Name())
			return
		}
		if v := o.String(); v != "bob" {
			t.Errorf("%s=%q, want %q", o.Name(), v, "bob")
		}
	})
	s.Parse([]string{"", "--the_name", "fred"})
	s.VisitAll(func(o getopt.Option) {
		if o.Name() != "--the_name" {
			t.Errorf("unexpected option found: %q", o.Name())
			return
		}
		if v := o.String(); v != "fred" {
			t.Errorf("%s=%q, want %q", o.Name(), v, "fred")
		}
	})
}

<<<<<<< HEAD
func TestRegister(t *testing.T) {
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Regiser did not panic on string")
			}
		}()
		Register("a")
	}()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Register did not panic on *string")
			}
		}()
		Register(new(string))
	}()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Registerdid not panic on bad tag")
			}
		}()
		register("test", &struct {
			F Flags `getopt:"bad"`
		}{}, getopt.New())
	}()
	if err := register("test", &struct {
		F Flags `encoding:"bob"`
	}{}, getopt.New()); err == nil {
		t.Errorf("Did not get an error on bad encoding")
=======
func TestSubRegisterAndParse(t *testing.T) {
	opts := struct {
		Value string `getopt:"--the_name=VALUE help"`
	}{
		Value: "bob",
	}

	for _, tt := range []struct {
		args  []string
		err   string
		value string
		out   []string
	}{{
		args:  []string{"name"},
		value: "bob",
		out: []string{},
	}, {
		args:  []string{"name", "-x"},
		err: "unknown option: -x",
		value: "bob",
	}, {
		args:  []string{"name","--the_name=fred"},
		value: "fred",
		out: []string{},
	}, {
		args:  []string{"name","--the_name=fred","a","b","c"},
		value: "fred",
		out:   []string{"a", "b", "c"},
	}} {
		myopts := opts
		args, err := SubRegisterAndParse(&myopts, tt.args)
		if s := check.Error(err, tt.err); s != "" {
			t.Errorf("%s", s)
			continue
		}
		if tt.value != myopts.Value {
			t.Errorf("%q got value %q, want %q", tt.args, myopts.Value, tt.value)
		}
		if !reflect.DeepEqual(tt.out, args) {
			t.Errorf("%q got args %q, want %q", tt.args, args, tt.out)
		}
>>>>>>> 67272c345e383137742e13808a8baead90629c4d
	}
}

func TestParseTag(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		tag  *optTag
		str  string
		err  string
	}{
		{
			name: "nothing",
		},
		{
			name: "dash",
			in:   "-",
		},
		{
			name: "dash-dash",
			in:   "--",
		},
		{
			name: "long arg",
			in:   "--option",
			str:  "{ --option }",
			tag: &optTag{
				long: "option",
			},
		},
		{
			name: "short arg",
			in:   "-o",
			str:  "{ -o }",
			tag: &optTag{
				short: 'o',
			},
		},
		{
			name: "long help",
			in:   "--option this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				long: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help1",
			in:   "--option -- this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				long: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help2",
			in:   "--option - this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				long: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help3",
			in:   "--option -- -this is an option",
			str:  `{ --option "-this is an option" }`,
			tag: &optTag{
				long: "option",
				help: "-this is an option",
			},
		},
		{
			name: "long and short arg",
			in:   "--option -o",
			str:  "{ --option -o }",
			tag: &optTag{
				long:  "option",
				short: 'o',
			},
		},
		{
			name: "short and long arg",
			in:   "-o --option",
			str:  "{ --option -o }",
			tag: &optTag{
				long:  "option",
				short: 'o',
			},
		},
		{
			name: "long arg with param",
			in:   "--option=PARAM",
			str:  "{ --option =PARAM }",
			tag: &optTag{
				long:  "option",
				param: "PARAM",
			},
		},
		{
			name: "short arg with param",
			in:   "-o=PARAM",
			str:  "{ -o =PARAM }",
			tag: &optTag{
				short: 'o',
				param: "PARAM",
			},
		},
		{
			name: "everything",
			in:   "--option=PARAM -o -- - this is help",
			str:  `{ --option -o =PARAM "- this is help" }`,
			tag: &optTag{
				long:  "option",
				short: 'o',
				param: "PARAM",
				help:  "- this is help",
			},
		},
		{
			name: "two longs",
			in:   "--option1 --option2",
			err:  "tag has too many long names",
		},
		{
			name: "two shorts",
			in:   "-a -b",
			err:  "tag has too many short names",
		},
		{
			name: "two parms",
			in:   "--option=PARAM1 -o=PARAM2",
			err:  "tag has multiple parameter names",
		},
		{
			name: "missing option",
			in:   "no option",
			err:  "tag missing option name",
		},
		{
			name: "long param only",
			in:   "--=PARAM",
			err:  "tag missing option name",
		},
		{
			name: "short param only",
			in:   "-=PARAM",
			err:  "tag missing option name",
		},
		{
			name: "two many dashes",
			in:   "---option",
			err:  "tag must not start with ---",
		},
		{
			name: "invalid short name",
			in:   "-short",
			err:  `getopt tag has invalid short name: "-short"`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := parseTag(tt.in)
			switch {
			case err == nil && tt.err != "":
				t.Fatalf("did not get expected error %v", tt.err)
			case err != nil && tt.err == "":
				t.Fatalf("unexpected error %v", err)
			case err == nil:
			case !strings.Contains(err.Error(), tt.err):
				t.Fatalf("got error %v, want %v", err, tt.err)
			}
			if !reflect.DeepEqual(tag, tt.tag) {
				t.Errorf("got %v, want %v", tag, tt.tag)
			}
			if tag != nil {
				str := tag.String()
				if str != tt.str {
					t.Errorf("%s: got string %q, want %q", tt.name, str, tt.str)
				}
			}
		})
	}
}

func TestArgPrefix(t *testing.T) {
	for _, tt := range []struct {
		in  string
		out string
	}{
		{"a", ""},
		{"-a", "-"},
		{"--a", "--"},
		{"", ""},
		{"-", "-"},
		{"--", "--"},
	} {
		if out := argPrefix(tt.in); out != tt.out {
			t.Errorf("argPrefix(%q) got %q want %q", tt.in, out, tt.out)
		}
	}
}

func TestDup(t *testing.T) {
	// Most of Dup is tested via other test methods.  We need to test the errors.
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Did not panic on string")
			}
		}()
		Dup("a")
	}()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Did not panic on *string")
			}
		}()
		Dup(new(string))
	}()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Did not panic on bad tag")
			}
		}()
		Dup(&struct {
			Opt bool `getopt:"bad tag"`
		}{})
	}()
}

func TestParse(t *testing.T) {
	args, cl := os.Args, getopt.CommandLine
	defer func() {
		os.Args, getopt.CommandLine = args, cl
	}()
	getopt.CommandLine = getopt.New()
	opts := &struct {
		Name string `geopt:"--name a name"`
	}{}
	Register(opts)
	os.Args = []string{"test", "--name", "bob", "arg"}
	pargs := Parse()
	if opts.Name != "bob" {
		t.Errorf("Got name %q, want %q", opts.Name, "bob")
	}
	if len(pargs) != 1 || pargs[0] != "arg" {
		t.Errorf("Got args %q, want %q", pargs, []string{"arg"})
	}
}
