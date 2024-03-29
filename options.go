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

// Package options provides a structured interface for getopt style flag
// parsing.  It is particularly helpful for parsing an option set more than once
// and possibly concurrently.  This package was designed to make option
// specification simpler and more concise.  It is a wrapper around the
// github.com/pborman/getopt/v2 package.
//
// Package options also provides a facility to specify command line options in a
// text file by using the Flags type (described below).
//
// # Option Decorations
//
// Options are declared in a structure that contains all information needed for
// the options.  Each exported field of the structure represents an option.  The
// fields tag is used to provide additional details.  The tag contains up to
// four pieces of information:
//
//	Long name of the option (e.g. --name)
//	Short name of the option (e.g., -n)
//	Parameter name (e.g. NAME)
//	Description (e.g., "Sets the name to NAME")
//
// The syntax of a tag is:
//
//	[--option[=PARAM]] [-o] [--] description
//
// The long and/or short options must come first in the tag.  The parameter name
// is specified by appending =PARAM to one of the declared options (e.g.,
// --option=VALUE).  The description is everything following the option
// declaration(s).  The options and description message are delimited by one or
// more white space characters.  An empty option (- or --) terminates option
// declarations, everything following is the description.  This enables the
// description to start with a -, e.g. "-v -- -v means verbose".
//
// # Example Tags
//
// The following are example tags
//
//	"--name=NAME -n sets the name to NAME"
//	"-n=NAME        sets the name to NAME"
//	"--name         sets the name"
//
// A tag of just "-" causes the field to be ignored an not used as an option.
// An empty tag or missing tag causes the tag to be auto-generated.
//
//	Name string -> "--name unspecified"
//	N int       -> "-n unspecified"
//
// # Types
//
// The fields of the structure can be any type that can be passed to getopt.Flag
// as a pointer (e.g., string, []string, int, bool, time.Duration, etc).  This
// includes any type that implements getopt.Value.
//
// # Example Structure
//
// The following structure declares 7 options and sets the default value of
// Count to be 42.  The --flags option is used to read option values from
// a file.
//
//	type theOptions struct {
//	    Flags   options.Flags `getopt:"--flags=PATH     read defaults from path"`
//	    Name    string        `getopt:"--name=NAME      name of the widget"`
//	    Count   int           `getopt:"--count -c=COUNT number of widgets"`
//	    Verbose bool          `getopt:"-v               be verbose"`
//	    N       int           `getopt:"-n=NUMBER        set n to NUMBER"`
//	    Timeout time.Duration `getopt:"--timeout        duration of run"`
//	    Lazy    string
//	}
//	var myOptions = theOptions {
//	    Count: 42,
//	}
//
// The help message generated from theOptions is:
//
//	Usage:  [-v] [-c COUNT] [--flags PATH] [--lazy value] [-n NUMBER] [--name NAME] [--timeout value] [parameters ...]
//	 -c, --count=COUNT    number of widgets
//	     --flags=PATH     read defaults from PATH
//	     --lazy=value     unspecified
//	 -n NUMBER            set n to NUMBER
//	     --name=NAME      name of the widget
//	     --timeout=value  duration of run
//	 -v                   be verbose
//
// # Usage
//
// The following are various ways to use the above declaration.
//
//	// Register myOptions, parse the command line, and set args to the
//	// remaining command line parameters
//	args := options.RegisterAndParse(&myOptions)
//
//	// Validate myOptions.
//	err := options.Validate(&myOptions)
//	if err != nil { ... }
//
//	// Register myOptions as command line options.
//	options.Register(&myOptions)
//
//	// Register myOptions as a new getopt Set.
//	set := getopt.New()
//	options.RegisterSet(&myOptions, set)
//
//	// Register a new instance of myOptions
//	vopts, set := options.RegisterNew(&myOptions)
//	opts := vopts.(*theOptions)
package options

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pborman/getopt/v2"
)

// Dup returns a shallow duplicate of i or panics.  Dup panics if i is not a
// pointer to struct or has an invalid getopt tag.  Dup does not copy
// non-exported fields or fields whose getopt tag is "-".
//
// Dup is normally used to create a unique instance of the set of options so i
// can be used multiple times.
func Dup(i interface{}) interface{} {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		panic(fmt.Errorf("%T is not a pointer to a struct", i))
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("%T is not a pointer to a struct", i))
	}
	t := v.Type()
	newi := reflect.New(t) // Same type as i
	ret := newi.Interface()
	newi = newi.Elem()

	n := t.NumField()
	for i := 0; i < n; i++ {
		field := t.Field(i)
		fv := newi.Field(i)
		tag := field.Tag.Get("getopt")
		if tag == "-" || !fv.CanSet() {
			continue
		}
		_, err := parseTag(tag)
		if err != nil {
			panic(err)
		}
		// Copy the value over
		fv.Set(v.Field(i))
	}
	return ret
}

// Register registers the fields in i with the standard command-line option set.
// It panics for the same reasons that RegisterSet panics.
func Register(i interface{}) {
	if err := register("", i, getopt.CommandLine); err != nil {
		panic(err)
	}
}

// RegisterAndParse and calls Register(i), getopt.Parse(), and returns
// getopt.Args().
func RegisterAndParse(i interface{}) []string {
	Register(i)
	getopt.Parse()
	return getopt.Args()
}

// SubRegisterAndParse is similar to RegisterAndParse except it is provided the
// arguments as args and on error the error is returned rather than written to
// standard error and the exiting the program.  This is done by creating a new
// getopt set, registering i with that set, and then calling Getopt on the set
// with args.
//
// SubRegisterAndParse is useful when you want to parse arguments other than
// os.Args (which is what RegisterAndParse does).
//
// The first element of args is equivalent to a command name and is not parsed.
//
// EXAMPLE:
//
//	func nameCommand(args []string) error {
//		opts := &struct {
//			Name string `getopt:"--name NAME the name to use"`
//		}{
//			Name: "none",
//		}
//		// If args does not include the subcommand name then prepend it
//		args = append([]string{"name"}, args...)
//
//		args, err := options.SubRegisterAndParse(opts, args)
//		if err != nil {
//			return err
//		}
//		fmt.Printf("The name is %s\n", opts.Name)
//		fmt.Printf("The parameters are: %q\n", args)
//	}
func SubRegisterAndParse(i interface{}, args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	set := getopt.New()
	if err := RegisterSet(args[0], i, set); err != nil {
		return nil, err
	}
	if err := set.Getopt(args, nil); err != nil {
		return nil, err
	}
	return set.Args(), nil
}

// Parse calls getopt.Parse and returns getopt.Args().
func Parse() []string {
	getopt.Parse()
	return getopt.Args()
}

// Validate validates i as a set of options or returns an error.
//
// Use Validate to assure that a later call to one of the Register functions
// will not panic.  Validate is typically called by an init function on
// structures that will be registered later.
func Validate(i interface{}) error {
	set := getopt.New()
	return register("", i, set)
}

// RegisterNew creates a new getopt Set, duplicates i, calls RegisterSet, and
// then returns them.  RegisterNew should be used when the options in i might be
// parsed multiple times requiring a new instance of i each time.
func RegisterNew(name string, i interface{}) (interface{}, *getopt.Set) {
	set := getopt.New()
	i = Dup(i)
	if err := register(name, i, set); err != nil {
		panic(err)
	}
	return i, set
}

// RegisterSet registers the fields in i, to the getopt Set set.  RegisterSet
// returns an error if i is not a pointer to struct, has an invalid getopt tag,
// or contains a field of an unsupported option type.  RegisterSet ignores
// non-exported fields or fields whose getopt tag is "-".
//
// If a Flags field is encountered, name is the name used to identify the set
// when parsing options.
//
// See the package documentation for a description of the structure to pass to
// RegisterSet.
func RegisterSet(name string, i interface{}, set *getopt.Set) error {
	return register(name, i, set)
}

func register(name string, i interface{}, set *getopt.Set) error {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("%T is not a pointer to a struct", i)
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("%T is not a pointer to a struct", i)
	}
	t := v.Type()

	n := t.NumField()
	for i := 0; i < n; i++ {
		field := t.Field(i)
		fv := v.Field(i)
		tag := field.Tag.Get("getopt")
		if tag == "-" || !fv.CanSet() {
			continue
		}
		o, err := parseTag(tag)
		if err != nil {
			panic(err)
		}
		if o == nil {
			n := strings.ToLower(field.Name)
			for x, r := range n {
				if x == 0 {
					o = &optTag{short: r}
				} else {
					o = &optTag{long: n}
					break
				}
			}
		}
		if o.help == "" {
			o.help = "unspecified"
		}
		hv := []string{o.help, o.param}
		if o.param == "" {
			hv = hv[:1]
		}
		opt := fv.Addr().Interface()
		if f, ok := opt.(*Flags); ok {
			f.Sets = append(f.Sets, Set{Name: name, Set: set})
			f.opt = set.FlagLong(opt, o.long, o.short, hv...)
			tag := field.Tag.Get("encoding")
			if tag == "" {
				tag = "simple"
			}
			decoderMu.Lock()
			decoder, ok := decoders[tag]
			decoderMu.Unlock()
			if !ok {
				return fmt.Errorf("unknown flags decoding type: %q", tag)
			}
			f.Decoder = decoder
		} else {
			op := set.FlagLong(opt, o.long, o.short, hv...)
			// Values that are of type bool are flags.
			if fv.Kind() == reflect.Bool {
				op.SetFlag()
			}
		}
	}
	return nil
}

// Lookup returns the value of the field in i for the specified option or nil.
// Lookup can be used if the structure declaring the options is not available.
// Lookup returns nil if i is invalid or does not have an option named option.
//
// # Example
//
// Fetch the verbose flag from an anonymous structure:
//
//	i, set := options.RegisterNew(&struct {
//		Verbose bool `getopt:"--verbose -v be verbose"`
//	})
//	set.Getopt(args, nil)
//	v := options.Lookup(i, "verbose").(bool)
func Lookup(i interface{}, option string) interface{} {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return nil
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()

	n := t.NumField()
	for i := 0; i < n; i++ {
		field := t.Field(i)
		fv := v.Field(i)
		tag := field.Tag.Get("getopt")
		if tag == "-" || !fv.CanSet() {
			continue
		}
		o, err := parseTag(tag)
		if err != nil {
			return nil
		}
		if o == nil {
			n := strings.ToLower(field.Name)
			for x, r := range n {
				if x == 0 {
					o = &optTag{short: r}
				} else {
					o = &optTag{long: n}
					break
				}
			}
		}
		if option == o.long || option == string(o.short) {
			return fv.Interface()
		}
	}
	return nil
}

// An optTag contains all the information extracted from a getopt tag.
type optTag struct {
	long  string
	short rune
	param string
	help  string
}

func (o *optTag) String() string {
	parts := make([]string, 0, 6)
	parts = append(parts, "{")
	if o.long != "" {
		parts = append(parts, "--"+o.long)
	}
	if o.short != 0 {
		parts = append(parts, "-"+string(o.short))
	}
	if o.param != "" {
		parts = append(parts, "="+o.param)
	}
	if o.help != "" {
		parts = append(parts, fmt.Sprintf("%q", o.help))
	}
	parts = append(parts, "}")
	return strings.Join(parts, " ")
}

// parseTag parses and returns tag as an optTag or returns an error.  nil, nil
// is returned if tag is empty or consists only of white space.
func parseTag(tag string) (*optTag, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, nil
	}
	next := tag
	var o optTag
	var arg, param string
	for {
		arg, param, next = nextOption(next)
		if arg == "" || arg == "-" || arg == "--" {
			if param != "" {
				// Only happens with "--=FOO" or "-=FOO"
				return nil, fmt.Errorf("getopt tag missing option name: %q", tag)
			}
			if o.long == "" && o.short == 0 {
				if next != "" {
					return nil, fmt.Errorf("getopt tag missing option name: %q", tag)
				}
				return nil, nil
			}
			o.help = next
			return &o, nil
		}
		if param != "" {
			if o.param != "" {
				return nil, fmt.Errorf("getopt tag has multiple parameter names: %q", tag)
			}
			o.param = param
		}
		switch argPrefix(arg) {
		case "-":
			if o.short != 0 {
				return nil, fmt.Errorf("getopt tag has too many short names: %q", tag)
			}
			for x, r := range arg[1:] {
				if x != 0 {
					return nil, fmt.Errorf("getopt tag has invalid short name: %q", tag)
				}
				o.short = r
			}
		case "--":
			if o.long != "" {
				return nil, fmt.Errorf("getopt tag has too many long names: %q", tag)
			}
			o.long = arg[2:]
		default:
			return nil, fmt.Errorf("getopt tag must not start with ---: %q", tag)
		}
	}
}

// nextOption returns the next option, optional parameter, and the rest of
// the string parsed from s.  If the option is "" then s does not start with
// an option (i.e., does not start with a -).
func nextOption(s string) (option, param, rest string) {
	if s == "" || s[0] != '-' {
		return "", "", s
	}
	if x := strings.Index(s, " "); x >= 0 {
		rest = strings.TrimSpace(s[x:])
		s = s[:x]
	}
	if x := strings.Index(s, "="); x >= 0 {
		return s[:x], s[x+1:], rest
	}
	return s, "", rest
}

// argPrefix returns the leading dashes in a.
func argPrefix(a string) string {
	for x, c := range a {
		if c != '-' {
			return a[:x]
		}
	}
	return a
}
