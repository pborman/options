package json

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	getopt "github.com/pborman/getopt/v2"
	"github.com/pborman/options"
)

func TestDecoder(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		out  map[string]interface{}
	}{
		{
			name: "empty",
			out:  map[string]interface{}{},
		},
		{
			name: "string",
			in: `
			{
				"key": "value"
			}`,
			out: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "number",
			in: `
			{
				"key": 42
			}`,
			out: map[string]interface{}{
				"key": json.Number("42"),
			},
		},
		{
			name: "multi-level",
			in: `
			{
				"name": "value",
				"child": {
					"key": 42
				}
			}`,
			out: map[string]interface{}{
				"name": "value",
				"child": map[string]interface{}{
					"key": json.Number("42"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := Decoder([]byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(out, tt.out) {
				t.Errorf("Got:\n%v\nWant:\n%v", out, tt.out)
			}
		})
	}
}

func mkFile(data string) (string, error) {
	tmpfile := fmt.Sprintf("%s/options_test.%s", os.TempDir(), uuid.New())
	return tmpfile, ioutil.WriteFile(tmpfile, []byte(data), 0644)
}

func TestParse(t *testing.T) {
	getopt.CommandLine = getopt.New()
	name := "fred"
	getopt.FlagLong(&name, "name", 'n')
	tmpfile, err := mkFile(`
{
    "name": "bob",
    "child": {
        "name": "jim"
    }
}
`)
	name2 := "john"
	s2 := getopt.New()
	s2.FlagLong(&name2, "name", 'n')

	defer os.Remove(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	f := options.NewFlags("flags")
	f.SetEncoding(Decoder)
	f.Sets = append(f.Sets, options.Set{Name: "child", Set: s2})
	if err := f.Set(tmpfile, nil); err != nil {
		t.Fatal(err)
	}
	if name != "bob" {
		t.Errorf("Got name %q, want %q", name, "bob")
	}
	if name2 != "jim" {
		t.Errorf("Got child.name %q, want %q", name2, "jim")
	}
}
