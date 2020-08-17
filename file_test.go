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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	getopt "github.com/pborman/getopt/v2"
)

func mkFile(data string) (string, error) {
	tmpfile := fmt.Sprintf("%s/options_test.%s", os.TempDir(), uuid.New())
	return tmpfile, ioutil.WriteFile(tmpfile, []byte(data), 0644)
}

// Type TM is a TextMartialer
type TM struct{ V string }

var tmErr = errors.New("tm error")

func (t *TM) MarshalText() ([]byte, error) {
	if t.V == "error" {
		return nil, tmErr
	}
	return []byte(":" + t.V), nil
}

func (t *TM) Set(value string, opt getopt.Option) error {
	t.V = value
	return nil
}
func (t *TM) String() string { return string(t.V) }

// Type S is a Stringer
type S struct{ V string }

func (s *S) String() string { return string(s.V) }
func (s *S) Set(value string, opt getopt.Option) error {
	s.V = value
	return nil
}

func TestFlags(t *testing.T) {
	type options struct {
		String   string        `getopt:"--string"`
		Int      int           `getopt:"--int"`
		Bool     bool          `getopt:"--bool"`
		Float    float64       `getopt:"--float"`
		Duration time.Duration `getopt:"--duration"`
		Flags    Flags         `getopt:"--flags"`
	}
	type suboptions struct {
		TM string `getopt:"--tm"`
		S  string `getopt:"-s"`
	}
	for _, tt := range []struct {
		name    string
		opts    *options
		flags   string
		args    []string
		want    *options
		wantsub *suboptions
	}{
		{
			name: "all",
			flags: `
				string = hello
				int = 42
				bool = true
				float = 4.2
				duration = 17s
				sub.tm = tmvalue
				sub.s = svalue
			`,
			want: &options{
				String:   "hello",
				Int:      42,
				Bool:     true,
				Float:    4.2,
				Duration: 17 * time.Second,
			},
			wantsub: &suboptions{
				TM: "tmvalue",
				S:  "svalue",
			},
		},
		{
			name: "no-override-before",
			flags: `
				string = hello
				int = 42
				bool = false
				float = 4.2
				duration = 17s
			`,
			args: []string{
				"--string=bob",
				"--int=17",
				"--float=1.7",
				"--duration=42s",
				"--bool",
				"--flags", "FLAGS",
			},
			want: &options{
				String:   "bob",
				Int:      17,
				Float:    1.7,
				Bool:     true,
				Duration: 42 * time.Second,
			},
		},
		{
			name: "no-override-after",
			flags: `
				string = hello
				int = 42
				bool = false
				float = 4.2
				duration = 17s
			`,
			args: []string{
				"--flags", "FLAGS",
				"--string=bob",
				"--int=17",
				"--float=1.7",
				"--duration=42s",
				"--bool",
			},
			want: &options{
				String:   "bob",
				Int:      17,
				Float:    1.7,
				Bool:     true,
				Duration: 42 * time.Second,
			},
		},
	} {
		if tt.opts == nil {
			tt.opts = &options{}
		}
		vopts, set := RegisterNew("", tt.opts)
		opts := vopts.(*options)

		var subopts *suboptions
		var subset *getopt.Set
		if tt.wantsub != nil {
			vopts, set := RegisterNew("sub", &suboptions{})
			subopts = vopts.(*suboptions)
			subset = set
		}
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := mkFile(tt.flags)
			defer os.Remove(tmpfile)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for i, a := range tt.args {
				if a == "FLAGS" {
					found = true
					tt.args[i] = tmpfile
				}
			}
			if !found {
				tt.args = append(tt.args, "--flags", tmpfile)
			}
			if subset != nil {
				opts.Flags.Sets = append(opts.Flags.Sets, Set{"sub", subset})
			}
			err = set.Getopt(append([]string{"test"}, tt.args...), nil)
			if err != nil {
				t.Fatal(err)
			}
			opts.Flags.Decoder = nil
			tt.want.Flags = opts.Flags
			if !reflect.DeepEqual(tt.want, opts) {
				t.Errorf("Got :\n%+v\nWant:\n%+v", opts, tt.want)
			}
			if subopts != nil {
				if !reflect.DeepEqual(tt.wantsub, subopts) {
					t.Errorf("Got :\n%+v\nWant:\n%+v", subopts, tt.wantsub)
				}
			}
		})
	}
}

func TestFlagsCommandLine(t *testing.T) {
	getopt.CommandLine = getopt.New()
	flags := &Flags{
		Sets:    []Set{{Set: getopt.CommandLine}},
		Decoder: SimpleDecoder,
	}
	tmpfile, err := mkFile(`name=bob`)
	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	var name string
	getopt.FlagLong(flags, "flags", 0)
	getopt.FlagLong(&name, "name", 'n')
	err = getopt.CommandLine.Getopt([]string{"test", "--flags", tmpfile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if name != "bob" {
		t.Errorf("Got name %q, want %q", name, "bob")
	}
}

func TestFlagsShortName(t *testing.T) {
	getopt.CommandLine = getopt.New()
	flags := &Flags{
		Sets:    []Set{{Set: getopt.CommandLine}},
		Decoder: SimpleDecoder,
	}
	tmpfile, err := mkFile(`n=bob`)
	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	getopt.FlagLong(flags, "flags", 0)
	getopt.FlagLong(&name, "name", 'n')
	err = getopt.CommandLine.Getopt([]string{"test", "--flags", tmpfile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if name != "bob" {
		t.Errorf("Got name %q, want %q", name, "bob")
	}
}

func TestFlagsIgnoreField(t *testing.T) {
	getopt.CommandLine = getopt.New()
	NewFlags("flags").IgnoreUnknown = true
	tmpfile, err := mkFile(`name=bob`)
	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	err = getopt.CommandLine.Getopt([]string{"test", "--flags", tmpfile}, nil)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestFlagsBadField(t *testing.T) {
	getopt.CommandLine = getopt.New()
	NewFlags("flags")
	tmpfile, err := mkFile(`name=bob`)
	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	err = getopt.CommandLine.Getopt([]string{"test", "--flags", tmpfile}, nil)
	if err == nil {
		t.Errorf("did not get error for unknown flags")
	}
}

func TestFlagsSet(t *testing.T) {
	getopt.CommandLine = getopt.New()
	name := "fred"
	getopt.FlagLong(&name, "name", 'n')
	tmpfile, err := mkFile(`name=bob`)
	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	NewFlags("flags").Set(tmpfile, nil)
	if name != "bob" {
		t.Errorf("Got name %q, want %q", name, "bob")
	}
}

func TestMissingFile(t *testing.T) {
	getopt.CommandLine = getopt.New()
	if err := NewFlags("flags").Set("?/this/file/does/not/exist", nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	getopt.CommandLine = getopt.New()
	if err := NewFlags("flags").Set("/this/file/does/not/exist", nil); err == nil {
		t.Error("did not get error for missing file")
	}
}

func TestTwoSets(t *testing.T) {
	getopt.CommandLine = getopt.New()
	name := "fred"
	getopt.FlagLong(&name, "name", 'n')
	tmpfile, err := mkFile(`
name=bob
child.name=jim
`)
	name2 := "john"
	s2 := getopt.New()
	s2.FlagLong(&name2, "name", 'n')

	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFlags("flags")
	f.Sets = append(f.Sets, Set{Name: "child", Set: s2})
	f.Set(tmpfile, nil)
	if name != "bob" {
		t.Errorf("Got name %q, want %q", name, "bob")
	}
	if name2 != "jim" {
		t.Errorf("Got child.name %q, want %q", name2, "jim")
	}
}

func TestExpand(t *testing.T) {
	os.Setenv("V1", "value1")
	os.Setenv("V2", "value2")
	os.Setenv("V3", "")
	for _, tt := range []struct {
		in  string
		out string
	}{
		{"", ""},
		{"abc", "abc"},

		{"$", "$"},
		{"$abc", "$abc"},
		{"${", "${"},
		{"${$", "${"},
		{"${abc", "${abc"},
		{"${$abc", "${abc"},
		{"${${abc", "${{abc"},
		{"${$$abc", "${$abc"},

		{"xyz$", "xyz$"},
		{"xyz${", "xyz${"},
		{"xyz${$", "xyz${"},
		{"xyz${abc", "xyz${abc"},
		{"xyz${$abc", "xyz${abc"},
		{"xyz${${abc", "xyz${{abc"},
		{"xyz${$$abc", "xyz${$abc"},
		{"xyz$abc", "xyz$abc"},

		{"${V1}", "value1"},
		{"${V2}", "value2"},
		{"${V3}", ""},
		{"${V1:-missing}", "value1"},
		{"${V3:-missing}", "missing"},
		{"${:-missing}", "missing"},
		{"${:-${}", "${"},
		{"${V1}${V2}${V3}", "value1value2"},
	} {
		out := expand(tt.in)
		if out != tt.out {
			t.Errorf("Expand(%q) got %q, want %q", tt.in, out, tt.out)
		}
	}
}

func testDecoder(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"tm": &TM{"tmvalue"},
		"s":  &S{"svalue"},
		"f":  42.0,
		"b":  true,
		"n":  false,
	}, nil
}

func TestDecoder(t *testing.T) {
	tmpfile, err := mkFile("bob")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile)
	type options struct {
		Flags Flags   `getopt:"--flags"`
		TM    TM      `getopt:"--tm"`
		S     S       `getopt:"-s"`
		F     float64 `getopt:"-f"`
		B     bool    `getopt:"-b"`
		N     bool    `getopt:"-n"`
	}
	vopts, set := RegisterNew("", &options{})
	opts := vopts.(*options)
	opts.Flags.SetEncoding(testDecoder)
	err = set.Getopt([]string{"test", "--flags", tmpfile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := opts.Flags
	opts.Flags = Flags{}
	want := &options{
		TM: TM{":tmvalue"},
		S:  S{"svalue"},
		F:  42.0,
		B:  true,
	}
	if !reflect.DeepEqual(want, opts) {
		t.Errorf("Got %v, want %v", opts, want)
	}
	vopts, set = RegisterNew("", &options{})
	opts = vopts.(*options)
	if err := f.Rescan("", set); err != nil {
		t.Fatal(err)
	}
	opts.Flags = Flags{}
	if !reflect.DeepEqual(want, opts) {
		t.Errorf("Got %v, want %v", opts, want)
	}
}

func TestDecoder2(t *testing.T) {
	tmpfile, err := mkFile("bob")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile)
	RegisterEncoding("testdecode", testDecoder)
	type options struct {
		Flags Flags   `getopt:"--flags" encoding:"testdecode"`
		TM    TM      `getopt:"--tm"`
		S     S       `getopt:"-s"`
		F     float64 `getopt:"-f"`
		B     bool    `getopt:"-b"`
		N     bool    `getopt:"-n"`
	}
	vopts, set := RegisterNew("", &options{})
	opts := vopts.(*options)
	err = set.Getopt([]string{"test", "--flags", tmpfile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	opts.Flags = Flags{}
	want := &options{
		TM: TM{":tmvalue"},
		S:  S{"svalue"},
		F:  42.0,
		B:  true,
	}
	if !reflect.DeepEqual(want, opts) {
		t.Errorf("Got %v, want %v", opts, want)
	}
}

func TestDecoderErr(t *testing.T) {
	tmpfile, err := mkFile("bob")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile)
	RegisterEncoding("testdecode", func(data []byte) (map[string]interface{}, error) {
		return map[string]interface{}{
			"v": struct{ A int }{A: 42},
		}, nil
	})
	type options struct {
		Flags Flags  `getopt:"--flags" encoding:"testdecode"`
		V     string `getopt:"-v"`
	}
	_, set := RegisterNew("", &options{})
	err = set.Getopt([]string{"test", "--flags", tmpfile}, nil)
	want := "struct { A int } not a string or number"
	if err == nil {
		t.Fatalf("did not get error %v", want)
	}
	if got := err.Error(); !strings.HasSuffix(got, want) {
		t.Errorf("Got error %q, want %q", got, want)
	}
}

func TestDecoderErr2(t *testing.T) {
	tmpfile, err := mkFile("bob")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile)
	RegisterEncoding("testdecode", func(data []byte) (map[string]interface{}, error) {
		return map[string]interface{}{
			"tm": &TM{"error"},
		}, nil
	})
	type options struct {
		Flags Flags  `getopt:"--flags" encoding:"testdecode"`
		TM    string `getopt:"--tm"`
	}
	_, set := RegisterNew("", &options{})
	err = set.Getopt([]string{"test", "--flags", tmpfile}, nil)
	rerr, ok := err.(*getopt.Error)
	if !ok {
		t.Errorf("Got error of type %T want %T", err, &getopt.Error{})
	}
	if rerr.Err != tmErr {
		t.Errorf("Got error %q, want %q", rerr.Err, tmErr)
	}
}

func TestDecoderErr3(t *testing.T) {
	tmpfile, err := mkFile("bob")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile)
	RegisterEncoding("testdecode", func(data []byte) (map[string]interface{}, error) {
		return nil, tmErr
	})
	type options struct {
		Flags Flags `getopt:"--flags" encoding:"testdecode"`
	}
	_, set := RegisterNew("", &options{})
	err = set.Getopt([]string{"test", "--flags", tmpfile}, nil)
	rerr, ok := err.(*getopt.Error)
	if !ok {
		t.Errorf("Got error of type %T want %T", err, &getopt.Error{})
	}
	got := rerr.Err.Error()
	want := tmErr.Error()
	if !strings.HasSuffix(got, want) {
		t.Errorf("Got error %q, want %q", rerr.Err, tmErr)
	}
}

func TestDecodeEmpty(t *testing.T) {
	tmpfile, err := mkFile("")
	if err != nil {
		t.Fatal(err)
	}
	type options struct {
		Flags Flags `getopt:"--flags" encoding:"testdecode"`
	}
	_, set := RegisterNew("", &options{})
	if err = set.Getopt([]string{"test", "--flags", tmpfile}, nil); err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestFlagsSetError(t *testing.T) {
	var f Flags

	if err := f.Set("", nil); err != nil {
		t.Errorf(`Flags.Set("") returned unexpected error %v`, err)
	}
	if err := f.Set("?", nil); err != nil {
		t.Errorf(`Flags.Set("?") returned unexpected error %v`, err)
	}
	if err := f.Set("flags", nil); err == nil {
		t.Errorf(`Flags.Set("flags") did not return an error`)
	} else if want := "options.Flags: not registered as an option"; err.Error() != want {
		t.Errorf(`Flags.Set("flags") got error %v, want %v`, err, want)
	}
	func() {
		type options struct {
			Flags Flags `getopt:"--flags"`
		}
		vopts, set := RegisterNew("", &options{})
		var s string
		opt := set.Flag(&s, 'x')
		opts := vopts.(*options)
		if err := opts.Flags.Set("flags", opt); err != nil {
			t.Errorf("Calling Flags.Set prior to parsing did not return nil: %v", err)
		}
	}()
}
