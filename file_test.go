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
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	getopt "github.com/pborman/getopt/v2"
)

func TestFlags(t *testing.T) {
	type options struct {
		String   string        `getopt:"--string"`
		Int      int           `getopt:"--int"`
		Bool     bool          `getopt:"--bool"`
		Float    float64       `getopt:"--float"`
		Duration time.Duration `getopt:"--duration"`
		Flags    Flags         `getopt:"--flags"`
	}
	for _, tt := range []struct {
		name  string
		opts  *options
		flags string
		args  []string
		want  *options
	}{
		{
			name: "all",
			flags: `{
				"string": "hello",
				"int": 42,
				"bool": true,
				"float": 4.2,
				"duration": "17s"
			}`,
			want: &options{
				String:   "hello",
				Int:      42,
				Bool:     true,
				Float:    4.2,
				Duration: 17 * time.Second,
			},
		},
		{
			name: "no-override-before",
			flags: `{
				"string": "hello",
				"int": 42,
				"bool": false,
				"float": 4.2,
				"duration": "17s"
			}`,
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
			flags: `{
				"string": "hello",
				"int": 42,
				"bool": false,
				"float": 4.2,
				"duration": "17s"
			}`,
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
		vopts, set := RegisterNew(tt.opts)
		opts := vopts.(*options)
		found := false
		for i, a := range tt.args {
			if a == "FLAGS" {
				found = true
				tt.args[i] = tt.flags
			}
		}
		if !found {
			tt.args = append(tt.args, "--flags", tt.flags)
		}
		t.Run(tt.name, func(t *testing.T) {
			err := set.Getopt(append([]string{"test"}, tt.args...), nil)
			if err != nil {
				t.Fatal(err)
			}
			tt.want.Flags = opts.Flags
			if !reflect.DeepEqual(tt.want, opts) {
				t.Errorf("Got :\n%+v\nWant:\n%+v", opts, tt.want)
			}
		})
	}
}

func TestFlagsCommandLine(t *testing.T) {
	getopt.CommandLine = getopt.New()
	flags := &Flags{
		Sets: []*getopt.Set{getopt.CommandLine},
	}
	var name string
	getopt.FlagLong(flags, "flags", 0)
	getopt.FlagLong(&name, "name", 'n')
	err := getopt.CommandLine.Getopt([]string{"test", "--flags", `{"name":"bob"}`}, nil)
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
		Sets: []*getopt.Set{getopt.CommandLine},
	}
	var name string
	getopt.FlagLong(flags, "flags", 0)
	getopt.FlagLong(&name, "name", 'n')
	err := getopt.CommandLine.Getopt([]string{"test", "--flags", `{"n":"bob"}`}, nil)
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
	err := getopt.CommandLine.Getopt([]string{"test", "--flags", `{"name":"bob"}`}, nil)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestFlagsBadField(t *testing.T) {
	getopt.CommandLine = getopt.New()
	NewFlags("flags")
	err := getopt.CommandLine.Getopt([]string{"test", "--flags", `{"name":"bob"}`}, nil)
	if err == nil {
		t.Errorf("did not get error for unknown flags")
	}
}

func TestFlagsSet(t *testing.T) {
	getopt.CommandLine = getopt.New()
	name := "fred"
	getopt.FlagLong(&name, "name", 'n')
	NewFlags("flags").Set(`{"name":"bob"}`, nil)
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

func TestFromFile(t *testing.T) {
	tmpfile := fmt.Sprintf("%s/options_test.%d", os.TempDir(), os.Getpid())
	defer os.Remove(tmpfile)
	if err := ioutil.WriteFile(tmpfile, []byte(`{"name":"bob"}`), 0644); err != nil {
		t.Fatal(err)
	}
	getopt.CommandLine = getopt.New()
	name := "fred"
	getopt.FlagLong(&name, "name", 'n')
	NewFlags("flags").Set(tmpfile, nil)
	if name != "bob" {
		t.Errorf("Got name %q, want %q", name, "bob")
	}
}
