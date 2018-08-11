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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/pborman/getopt/v2"
)

// A Flags is an getopt.Value that accepts JSON encoded flags.  If set to a
// string that starts with "{" then the string is assumed to be a JSON encoded
// set of values, otherwise the string is the pathname of a file with a JSON
// encoded set of string.  If the string starts with a ? then the rest of the
// string is the pathname to load and it is not an error if the file does not
// exist.
//
// The value in the JSON encoded data should look something like:
//
//	{
//		"name": "bob",
//		"v": true,
//		"n": 42
//	}
//
// This blob is the equivalent of "--name=bob -v -n=42" except the values in the
// blob will never override an option value that is set on the command line.
// The following command lines all set "name" to "bob" assuming --flags is of
// type Flags:
//
//	--flags '{"name": "bob"}'
//	--flags '{"name": "fred"}' --name bob
//	--name bob --flags '{"name": "fred"}'
//
// The options in the getopt.Sets listed will be updated. If the same option
// name occurs in two sets, the value is only set in the first set.
//
// A Flags in an options structure automatically has the options structure set
// appended.  When registered with getopt.Flag, it is the caller's
// responsibility to set Sets.
//
// To use a Flags as a flag in a options structure, include a field of type
// Flags in the structure, such as:
//
//	Flags options.Flags `getopt:"--flags json encoded command line parameters"
//
// To use a Flags with the standard command line set:
//
//	options.NewFlags("flags")
//
// Unless IgnoreUnknown is set, it is an error to pass in a JSON blob that
// references an unknown option.
type Flags struct {
	Sets          []*getopt.Set
	IgnoreUnknown bool
	path          string
	opt           getopt.Option
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
	json := &Flags{Sets: []*getopt.Set{getopt.CommandLine}}
	json.opt = getopt.FlagLong(json, name, 0, "json encoded command line parameters")
	return json
}

// Set implements getopt.Value.  Set can be called directly by passing a nil
// getopt.Option.  Set is a no-op if value is the empty string.  This can be
// used to read values from the environment:
//
//	options.NewFlags("flags").Set(os.GetEnv("MY_PROGRAM_FLAGS"), nil)
func (f *Flags) Set(value string, opt getopt.Option) error {
	// We trim spaces incase someone says:
	//	--flags '
	//		{ "key": "value" }
	//	'
	value = strings.TrimSpace(value)
	if value == "" {
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
	var err error
	var data []byte

	// If the value starts with '{' assume it is a json blob.
	// Otheriwse it is a path name.
	// close the } above

	switch value[0] {
	case '{': // }
		data = []byte(value)
	case '?':
		value = value[1:]
		data, err = ioutil.ReadFile(value)
		if err != nil {
			return nil
		}
	default:
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
	m := map[string]interface{}{}
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()
	for decoder.More() {
		if err := decoder.Decode(&m); err != nil {
			return err
		}
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for _, set := range f.Sets {
		var err error
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

			var s string
			switch v := v.(type) {
			case string:
				s = v
			case float64:
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
	if f.IgnoreUnknown || len(m) == 0 {
		return nil
	}
	names := make([]string, 1, len(m)+1)
	if value[0] == '{' { // }
		names[0] = "unknown flags:"
	} else {
		names[0] = value + ": unknown flags:"
	}
	for k := range m {
		names = append(names, "--"+k)
	}
	sort.Strings(names[1:])
	return errors.New(strings.Join(names, "\n    "))
}

// String implements getopt.Value.
func (f *Flags) String() string {
	return f.path
}
