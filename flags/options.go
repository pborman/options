// Copyright 2023 Paul Borman
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

// Package flags is a simplified version github.com/pborman/options that works
// with the standard flag package and possibly other flag packages.
//
// Package flags provides a structured interface for flag parsing.  It is
// particularly helpful for parsing an option set more than once and possibly
// concurrently.  This package was designed to make option specification simpler
// and more concise.  It is a wrapper around the the standard flag pacakge.
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
//	[[-]-option[=PARAM]] [--] description
//
// The option must come first in the tag.  It is prefixed by "-" or "--".  The
// parameter name is specified by appending =PARAM to one of the declared
// options (e.g., --option=VALUE).  The description is everything following the
// option declaration(s).  The options and description message are delimited by
// one or more white space characters.  An empty option (- or --) terminates
// option declarations, everything following is the description.  This enables
// the description to start with a -, e.g. "-v -- -v means verbose".
//
// # Example Tags
//
// The following are example tags
//
//	"--name=NAME sets the name to NAME"
//	"-n=NAME     sets the name to NAME"
//	"--name      sets the name"
//
// A tag of just "-" causes the field to be ignored an not used as an option.
// An empty tag or missing tag causes the tag to be auto-generated.
//
//	Name string -> "--name unspecified"
//	N int       -> "-n unspecified"
//
// # Types
//
// The fields of the structure must be compatible with one of the folllowing
// types:
//	bool
//	int
//	int64
//	float64
//	string
//	uint
//	uint64
//	[]string
//	Value
//	time.Duration
//
// # Example Structure
//
// The following structure declares 7 options and sets the default value of
// Count to be 42.
//
//	type theOptions struct {
//	    Name    string        `getopt:"--name=NAME   name of the widget"`
//	    Count   int           `getopt:"--count=COUNT number of widgets"`
//	    Verbose bool          `getopt:"-v            be verbose"`
//	    N       int           `getopt:"-n=NUMBER     set n to NUMBER"`
//	    Timeout time.Duration `getopt:"--timeout     duration of run"`
//	    List    []string      `getopt:"--list=ITEM   add ITEM to the list"`
//	    Lazy    string
//	}
//	var myOptions = theOptions {
//	    Count: 42,
//	}
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
//	set := flag.NewFlagSet("", flag.ExitOnError)
//	options.RegisterSet(&myOptions, set)
//
//	// Register a new instance of myOptions
//	vopts, set := options.RegisterNew(&myOptions)
//	opts := vopts.(*theOptions)
package flags

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

// Value is the interface to the dynamic value stored in a flag. (The default
// value is represented as a string.) Value is a copy of flag.Value
type Value interface {
	String() string
	Set(string) error
}

// NewFlagSet and CommandLine can be replaced to use a different flag package.
// They default to the standard flag package.
var (
	NewFlagSet  = func(name string) FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }
	CommandLine FlagSet = flag.CommandLine
)

// A FlagSet implements a set of flags.  flag.FlagSet from the standard flag package implements FlagSet.
// The FlagSet must also have the method:
//
//	Var(v valueType, name, usage string)
//
// Where valueType implements the Value interface (which flag.Value does).
// We cannot put Var in the interface due to the Value type.
type FlagSet interface {
	Parse([]string) error
	Args() []string
	NArg() int
	DurationVar(p *time.Duration, name string, value time.Duration, usage string)
	StringVar(p *string, name string, value string, usage string)
	IntVar(p *int, name string, value int, usage string)
	Int64Var(p *int64, name string, value int64, usage string)
	UintVar(p *uint, name string, value uint, usage string)
	Uint64Var(p *uint64, name string, value uint64, usage string)
	Float64Var(p *float64, name string, value float64, usage string)
	BoolVar(p *bool, name string, value bool, usage string)
}

type list []string

func (l *list) Set(s string) error {
	*l = append(*l, s)
	return nil
}

func (l *list) String() string {
	return strings.Join(*l, " ")
}

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
	if err := register("", i, CommandLine); err != nil {
		panic(err)
	}
}

// RegisterAndParse and calls Register(i), flag.Parse(), and returns
// getopt.Args().
func RegisterAndParse(i interface{}) []string {
	Register(i)
	CommandLine.Parse(os.Args[1:])
	return CommandLine.Args()
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
	set := NewFlagSet("")
	if err := RegisterSet(args[0], i, set); err != nil {
		return nil, err
	}
	if err := set.Parse(args[1:]); err != nil {
		return nil, err
	}
	return set.Args(), nil
}

// Parse calls flag.Parse and returns flag.Args().
func Parse() []string {
	CommandLine.Parse(os.Args[1:])
	return CommandLine.Args()
}

// Validate validates i as a set of options or returns an error.
//
// Use Validate to assure that a later call to one of the Register functions
// will not panic.  Validate is typically called by an init function on
// structures that will be registered later.
func Validate(i interface{}) error {
	set := NewFlagSet("")
	return register("", i, set)
}

// RegisterNew creates a new flag.FlagSet, duplicates i, calls RegisterSet, and
// then returns them.  RegisterNew should be used when the options in i might be
// parsed multiple times requiring a new instance of i each time.
func RegisterNew(name string, i interface{}) (interface{}, FlagSet) {
	set := NewFlagSet("")
	i = Dup(i)
	if err := register(name, i, set); err != nil {
		panic(err)
	}
	return i, set
}

// RegisterSet registers the fields in i, to the flag.FlagSet set.  RegisterSet
// returns an error if i is not a pointer to struct, has an invalid getopt tag,
// or contains a field of an unsupported option type.  RegisterSet ignores
// non-exported fields or fields whose getopt tag is "-".
//
// If a Flags field is encountered, name is the name used to identify the set
// when parsing options.
//
// See the package documentation for a description of the structure to pass to
// RegisterSet.
func RegisterSet(name string, i interface{}, set FlagSet) error {
	return register(name, i, set)
}

func register(name string, i interface{}, set FlagSet) error {
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
			o = &optTag{name: strings.ToLower(field.Name)}
		}
		if o.help == "" {
			o.help = "unspecified"
		}
		hv := []string{o.help, o.param}
		if o.param == "" {
			hv = hv[:1]
		}
		opt := fv.Addr().Interface()
		switch t := opt.(type) {
		case Value:
			setvar(set, t, o.name, o.help)
		case *[]string:
			setvar(set, (*list)(t), o.name, o.help)
		case *time.Duration:
			set.DurationVar(t, o.name, *t, o.help)
		case *string:
			set.StringVar(t, o.name, *t, o.help)
		case *int:
			set.IntVar(t, o.name, *t, o.help)
		case *int64:
			set.Int64Var(t, o.name, *t, o.help)
		case *uint:
			set.UintVar(t, o.name, *t, o.help)
		case *uint64:
			set.Uint64Var(t, o.name, *t, o.help)
		case *float64:
			set.Float64Var(t, o.name, *t, o.help)
		case *bool:
			set.BoolVar(t, o.name, *t, o.help)
		default:
			panic(fmt.Sprintf("invalid option type: %T", fv.Interface()))
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
//	set.Parse(args)
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
			o = &optTag{name: strings.ToLower(field.Name)}
		}
		if option == o.name {
			return fv.Interface()
		}
	}
	return nil
}

// An optTag contains all the information extracted from a getopt tag.
type optTag struct {
	name  string
	param string
	help  string
}

func (o *optTag) String() string {
	parts := make([]string, 0, 6)
	parts = append(parts, "{")
	switch len(o.name) {
	case 0:
	case 1:
		parts = append(parts, "-"+o.name)
	default:
		parts = append(parts, "--"+o.name)
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
			if o.name == "" {
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
		if o.name != "" {
			return nil, fmt.Errorf("getopt tag has too many names: %q", tag)
		}
		// Strip off the leading -- or -.
		o.name = strings.TrimPrefix(arg[1:], "-")
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

var (
	valueInterface Value
	valueType      = reflect.TypeOf(&valueInterface).Elem()
	stringType     = reflect.TypeOf("")
)

// setvar calls fs.Var(value, name, usage).  The value type Var expects must
// implement the Value inteface.  This enables this package to pass a Value
// where the flag package expected a flag.Value.  An error is returned
// if fs does not have a Var method with an appropriate signature.
func setvar(fs interface{}, value Value, name, usage string) error {
	v := reflect.ValueOf(fs)
	m := v.MethodByName("Var")
	if !m.IsValid() {
		return fmt.Errorf("Type %v missing Var method", v.Type())
	}
	t := m.Type()
	// We want to be able to call Var(value, name, usage)
	if t.NumIn() != 3 || t.NumOut() != 0 || !t.In(0).Implements(valueType) || !t.In(1).AssignableTo(stringType) || !t.In(1).AssignableTo(stringType) {
		return fmt.Errorf("Type %v has the wrong signature for Var", v.Type())
	}
	m.Call([]reflect.Value{
		reflect.ValueOf(value),
		reflect.ValueOf(name),
		reflect.ValueOf(usage),
	})
	return nil
}
