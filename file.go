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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/pborman/getopt/v2"
)

// A Flags is an getopt.Value that reads initial command line flags from a file
// named by the flags value.  The flags read from the file are effectively read
// prior to any other command line flag.  If a flag is set both in a flags file
// and on the command line directly, the command line value is the value that is
// used.
//
// It is an error if the specified file does not exist unless the pathname is
// prefixed with a ? (the ? is stripped), e.g., --flags=?my-flags.
//
// The format of the flags file can be specified by either using the
// SetEncoding method or by using the "encoding" struct Flags field tag.
//
// The default file encoding is provided by the SimpleDecoder function
// (registered as the encoding "simple").  Other file encodings can be specified
// by either using the SetEncoding or supplying the encoding tag in options
// structure (see below for more details).
//
// Consider the file name my-flags:
//
//	name = bob
//	v = true
//	n = 42
//
// Passing --flags=my-flags is the equivalent of prefacing the command line
// argumebts with "--name=bob -v -n=42".  Below are are example command lines
// and the resulting value of name:
//
//	--flags my-flags                # name is bob
//	--flags my-flags --name fred    # name is fred
//	--name fred --flags my-flags    # name is fred
//
// The Sets field specifies the options that the read flags modify.  If the same
// flag appears in two sets, only the first set is modified.  The default
// getopt.Set is a single element of either getopt.CommandLine or the getopt.Set
// passed to RegisterSet or returned by RegisterNew.
//
// The encoding can be changed from SimpleDecoder, a.k.a. "simple" by either
// using the SetEncoding method or by specifying the registered encoding as
// a struct tag to the Flags field in an options structure, e.g.:
//
//	Flags options.Flags `getopt:"--flags specify flags file" encoding:"json"`
//
// (Importing the package github.com/pborman/options/json registers the json
// encoding.)
//
// Unless IgnoreUnknown is set, it is an error to pass in a JSON blob that
// references an unknown option.
type Flags struct {
	Sets          []Set
	IgnoreUnknown bool
	Decoder       FlagsDecoder
	path          string
	opt           getopt.Option
	m             map[string]interface{}
}

var (
	decoderMu sync.Mutex
	decoders  = map[string]FlagsDecoder{"simple": SimpleDecoder}
)

// A FlagsDecoder the data in bytes as a set of key value pairs.  The values
// must be type assertable to a strconv.TextMarshaller, a fmt.Stringer, a
// string, a bool, or one of the non-complex numeric types (e.g., int).
type FlagsDecoder func([]byte) (map[string]interface{}, error)

func RegisterEncoding(name string, dec FlagsDecoder) {
	decoderMu.Lock()
	decoders[name] = dec
	decoderMu.Unlock()
}

// NewFlags returns a new Flags registered on the standard CommandLine as a long
// named option.
//
// Typical usage:
//
//	options.NewFlags("flags")
//
// To ignore unknown flag names:
//
//	options.NewFlags("flags").IgnoreUnknown = true
func NewFlags(name string) *Flags {
	flags := &Flags{
		Sets:    []Set{{Set: getopt.CommandLine}},
		Decoder: SimpleDecoder,
	}
	flags.opt = getopt.FlagLong(flags, name, 0, "file containing command line parameters")
	return flags
}

// A Set is a named getopt.Set.
type Set struct {
	Name string
	*getopt.Set
}

// SetEncoding returns f after setting the decoding function to decoder.
// For example:
//
//	flags := options.NewFlags("flags").SetEncoding(json.Decoder)
func (f *Flags) SetEncoding(decoder FlagsDecoder) *Flags {
	f.Decoder = decoder
	return f
}

// rescanFlags is the magic path name passed to set to cause it to
// re-scan options but not read a file.
var rescanFlags = string("\000\000\000")

// Set implements getopt.Value.  Set can be called directly by passing a nil
// getopt.Option.  Set is a no-op if value is the empty string.  Set does
// simple environment variable expansion on value.
//
// The expansion forms ${NAME} and ${NAME:-VALUE} are supported.  In the latter
// case VALUE will be used if NAME is not found or set to the empty string.
// Use "${$" to represent a literal "${".
//
//	var myOptions struct {
//		...
//		Flags options.Flags `getopt:"--flags specify flags file"`
//	}{}
//	func init() {
//		options.Register(&myOptions)
//		options.Flags.Set("?${HOME}/.my.flags", nil)
//	}
//
// or
//
//	options.NewFlags("flags").Set("?${HOME}/.my.flags", nil)
func (f *Flags) Set(value string, opt getopt.Option) error {
	value = expand(value)
	if value == "" || value == "?" {
		return nil
	}
	if opt == nil {
		opt = f.opt
		if opt == nil {
			return errors.New("options.Flags: not registered as an option")
		}
	} else if !opt.Seen() {
		return nil
	}

	if value == rescanFlags {
		value = f.path
	} else {
		var data []byte
		var err error

		switch value[0] {
		case '?': // okay for the file
			value = value[1:]
			data, err = ioutil.ReadFile(value)
			if err != nil {
				return nil
			}
		default: // filename
			data, err = ioutil.ReadFile(value)
			if err != nil {
				return err
			}
		}

		f.path = value
		data = bytes.TrimSpace(data)
		if len(data) == 0 {
			return nil
		}

		// We may get set multiple times, for example, a defaults file
		// and then a file specified by --flags.  We might also have a
		// map that contains subsets of flags that we don't know about
		// yet.  By keeping the merged list of options that we have seen
		// we can re-play after the subset is registered.
		m, err := f.Decoder(data)
		if err != nil {
			return fmt.Errorf("%s: %v", value, err)
		}
		f.m = mergemap(f.m, m)
	}

	// Now make a duplicate to work with.
	m := mergemap(nil, f.m)

	// matched is the names of subsets that we found
	matched := map[string]bool{}
	for _, set := range f.Sets {
		var err error
		// So we don't forget the original map
		m := m
		matched[set.Name] = true
		if set.Name != "" {
			switch sm := m[set.Name].(type) {
			case nil:
				continue
			case map[string]interface{}:
				m = sm
			default:
				continue
			}
		}
		set.VisitAll(func(o getopt.Option) {
			if err != nil {
				return
			}
			var v interface{}
			var ok bool
			n := o.LongName()
			if n != "" {
				v, ok = m[n]
			}
			if !ok {
				n = o.ShortName()
				if n != "" {
					v, ok = m[n]
				}
			}
			if !ok {
				return
			}
			delete(m, n)

			type Stringer interface {
				String() string
			}
			type TextMarshaler interface {
				MarshalText() (text []byte, err error)
			}

			var s string
			switch v := v.(type) {
			case TextMarshaler:
				data, err := v.MarshalText()
				if err != nil {
					return
				}
				s = string(data)
			case Stringer:
				s = v.String()
			case string:
				s = v
			case float64, float32,
				int, int64, int32, int16, int8,
				uint, uint64, uint32, uint16, uint8:
				s = fmt.Sprintf("%v", v)
			case bool:
				if v {
					s = "true"
				} else {
					s = "false"
				}
			default:
				err = fmt.Errorf("%s: %q not a string or number", value, n)
				return
			}
			// Don't override set values
			if o.Seen() {
				return
			}
			o.Value().Set(s, o)
		})
		if err != nil {
			return err
		}
	}

	if f.IgnoreUnknown {
		return nil
	}

	// Determine if there are any unknown global flags or flags for this
	// particular sub-command.  We ignore all other sets of flags.
	names := make([]string, 1, len(m)+1)
	names[0] = fmt.Sprintf("%s: unrecognized flags:", value)
	for k, v := range m {
		// TODO(borman): are we handling suboptions correctly here?
		// if !matched[k] {
		// 	continue
		// }
		sm, ok := v.(map[string]interface{})
		if !ok {
			names = append(names, "--"+k)
			continue
		}
		for sk := range sm {
			names = append(names, "--"+k+"."+sk)
		}
	}
	if len(names) == 1 {
		return nil
	}
	sort.Strings(names[1:])
	return errors.New(strings.Join(names, "\n    "))
}

// Rescan sets values in set from the values previously set in f.
func (f *Flags) Rescan(name string, set *getopt.Set) error {
	osets := f.Sets
	defer func() { f.Sets = osets }()
	f.Sets = []Set{{
		Name: name,
		Set:  set,
	}}
	return f.Set(rescanFlags, nil)

}

// String implements getopt.Value.
func (f *Flags) String() string {
	return f.path
}

// mergemap merges the entries in old into new and returns new.  If new is
// nil then a new map is created.
func mergemap(new, old map[string]interface{}) map[string]interface{} {
	if new == nil {
		new = map[string]interface{}{}
	}
	for k, v := range old {
		if vm, ok := v.(map[string]interface{}); ok {
			v = mergemap(nil, vm)
		}
		new[k] = v
	}
	return new
}

// expand does simple ${VALUE} variable expansion on s and returns the result.
// It supports ${NAME} and ${NAME:-VALUE}.  If VALUE is provided then it is used
// if NAME is either empty or not set.  User "${$" to represent a literal "${".
func expand(s string) string {
	var parts []string
	for {
		x := strings.Index(s, "${") // }
		if x < 0 || x+2 == len(s) {
			return strings.Join(append(parts, s), "")
		}
		if s[x+2] == '$' {
			parts = append(parts, s[:x+2])
			s = s[x+3:]
			continue
		}
		parts = append(parts, s[:x])
		s = s[x+2:]
		// {
		x = strings.Index(s, "}")
		if x < 0 {
			return strings.Join(append(parts, "${", s), "") // }
		}
		var name, value string
		name = s[:x]
		s = s[x+1:]
		if x := strings.Index(name, ":-"); x >= 0 {
			value = name[x+2:]
			name = name[:x]
		}
		if env := os.Getenv(name); env != "" {
			value = env
		}
		parts = append(parts, value)
	}
}
