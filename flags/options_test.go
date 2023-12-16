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

package flags

import (
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/gnmi/errdiff"
)

type X string
func (x *X)Set(s string) error { *x = X(s); return nil }
func (x *X)String() string { return (string)(*x) }

func TestVar(t *testing.T) {
	var x X
	fs := flag.NewFlagSet("v", flag.ExitOnError)
	if err := setvar(fs, &x, "flag", "usage"); err != nil {
		t.Fatal(err)
	}
	found := false
	fs.VisitAll(func(f *flag.Flag) {
		if f.Name == "flag" {
			found = true
		}
	})
	if !found {
		t.Errorf("flag was not set")
	}
}

func TestLookup(t *testing.T) {
	opt := &struct {
		Ignore bool   `getopt:"-"`
		Option string `getopt:"--option"`
		Lazy   string
	}{
		Option: "value",
		Lazy:   "lazy",
	}
	if o := Lookup(opt, "option"); o.(string) != "value" {
		t.Errorf("--option returned value %q, want %q", o, "value")
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
		Option  string `getopt:"--option"`
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

func TestRegisterSet(t *testing.T) {
	opts := &struct {
		Name string `getopt:"--the_name"`
	}{
		Name: "bob",
	}
	s := NewFlagSet("")
	RegisterSet("", opts, s)
	s.(*flag.FlagSet).VisitAll(func(f *flag.Flag) {
		if f.Name != "the_name" {
			t.Errorf("unexpected option found: %q", f.Name)
			return
		}
		if v := f.Value.String(); v != "bob" {
			t.Errorf("%s=%q, want %q", f.Name, v, "bob")
		}
	})
	s.Parse([]string{"--the_name", "fred"})
	s.(*flag.FlagSet).VisitAll(func(f *flag.Flag) {
		if f.Name != "the_name" {
			t.Errorf("unexpected option found: %q", f.Name)
			return
		}
		if v := f.Value.String(); v != "fred" {
			t.Errorf("%s=%q, want %q", f.Name, v, "fred")
		}
	})
}

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
			F int `getopt:"bad"`
		}{}, NewFlagSet(""))
	}()
}

func TestMultiString(t *testing.T) {
	var opts struct {
		Value []string `getopt:"--multi=VALUE help"`
		List  []string    `getopt:"--list=VALUE help"`
	}
	_, err := SubRegisterAndParse(&opts, []string{"name", "--multi", "value1", "--multi", "value2", "--list", "item1", "--list", "item2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Value) != 2 {
		t.Errorf("got %d values, want 2", len(opts.Value))
	} else {
		if opts.Value[0] != "value1" {
			t.Errorf("got %s, want value1", opts.Value[0])
		}
		if opts.Value[1] != "value2" {
			t.Errorf("got %s, want value2", opts.Value[1])
		}
	}
	if len(opts.List) != 2 {
		t.Errorf("got %d values, want 2", len(opts.List))
	} else {
		if opts.List[0] != "item1" {
			t.Errorf("got %s, want item1", opts.List[0])
		}
		if opts.List[1] != "item2" {
			t.Errorf("got %s, want item2", opts.List[1])
		}
	}
}

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
	}, {
		args:  []string{"name", "--the_name=fred"},
		value: "fred",
	}, {
		args:  []string{"name", "--the_name=fred", "a", "b", "c"},
		value: "fred",
		out:   []string{"a", "b", "c"},
	}} {
		myopts := opts
		args, err := SubRegisterAndParse(&myopts, tt.args)
		if s := errdiff.Check(err, tt.err); s != "" {
			t.Errorf("%s", s)
			continue
		}
		if len(args) == 0 {
			args = nil
		}
		if tt.value != myopts.Value {
			t.Errorf("%q got value %q, want %q", tt.args, myopts.Value, tt.value)
		}
		if !reflect.DeepEqual(tt.out, args) {
			t.Errorf("%q got args %#v, want %#v", tt.args, args, tt.out)
		}
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
				name: "option",
			},
		},
		{
			name: "short arg",
			in:   "-o",
			str:  "{ -o }",
			tag: &optTag{
				name: "o",
			},
		},
		{
			name: "long help",
			in:   "--option this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help1",
			in:   "--option -- this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help2",
			in:   "--option - this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help3",
			in:   "--option -- -this is an option",
			str:  `{ --option "-this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "-this is an option",
			},
		},
		{
			name: "long arg with param",
			in:   "--option=PARAM",
			str:  "{ --option =PARAM }",
			tag: &optTag{
				name:  "option",
				param: "PARAM",
			},
		},
		{
			name: "everything",
			in:   "--option=PARAM -- - this is help",
			str:  `{ --option =PARAM "- this is help" }`,
			tag: &optTag{
				name:  "option",
				param: "PARAM",
				help:  "- this is help",
			},
		},
		{
			name: "two longs",
			in:   "--option1 --option2",
			err:  "tag has too many names",
		},
		{
			name: "two shorts",
			in:   "-a -b",
			err:  "tag has too many names",
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
	args, cl := os.Args, flag.CommandLine
	defer func() {
		os.Args, flag.CommandLine = args, cl
	}()
	CommandLine = NewFlagSet("")
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
