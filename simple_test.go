package options

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnescape(t *testing.T) {
	for _, tt := range []struct {
		in, out string
	}{
		{``, ``},
		{`# comment`, ``},
		{`a word`, `a word`},
		{` leading space`, `leading space`},
		{`trailing space `, `trailing space`},
		{`  space  `, `space`},
		{`a \# pound # and comment`, `a # pound`},
		{`name = "value " `, `name = "value "`},
		{`\\`, `\`},
		{`\#`, `#`},
		{`\\\#`, `\#`},
		{`\\\#\x`, `\#x`},
		{`foo\`, `foo`},
	} {
		out := unescape([]byte(tt.in))
		if out != tt.out {
			t.Errorf("`%s`: got `%s`, want `%s`", tt.in, out, tt.out)
		}
	}
}

func TestSimpleDecoder(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		m    map[string]interface{}
		err  string
	}{
		{
			name: `empty`,
			m:    map[string]interface{}{},
		},
		{
			in: `# comment`,
			m:  map[string]interface{}{},
		},
		{
			in: `name=value`,
			m:  map[string]interface{}{`name`: `value`},
		},
		{
			in: `name = value`,
			m:  map[string]interface{}{`name`: `value`},
		},
		{
			in: ` name = value `,
			m:  map[string]interface{}{`name`: `value`},
		},
		{
			in: `name = "value"`,
			m:  map[string]interface{}{`name`: `value`},
		},
		{
			in: `name = "value "`,
			m:  map[string]interface{}{`name`: `value `},
		},
		{
			in: `name=value #comment`,
			m:  map[string]interface{}{`name`: `value`},
		},
		{
			name: "missing value",
			in:   `name`,
			err:  `missing value: "name"`,
		},
		{
			name: "space in name",
			in:   `a name = a value`,
			err:  `space in name: "a name = a value"`,
		},
		{
			name: "missing name",
			in:   `=value`,
			err:  `missing name: "=value"`,
		},
		{
			name: "complex",
			in: `
# This is a multiple line test
key1=value1
  key2 = "value 2" # comment
key3 = "value #" # the comment wasn't escaped
`,
			m: map[string]interface{}{
				"key1": `value1`,
				"key2": `value 2`,
				"key3": `"value`,
			},
		},
	} {
		if tt.name == "" {
			tt.name = tt.in
		}
		t.Run(tt.name, func(t *testing.T) {
			m, err := SimpleDecoder([]byte(tt.in))
			switch {
			case err == nil && tt.err == "":
			case err == nil:
				t.Fatalf("did not get expected error %v", tt.err)
			case tt.err == "":
				t.Fatalf("unexpected error %v", err)
			case !strings.Contains(err.Error(), tt.err):
				t.Fatalf("got error %v, want %v", err, tt.err)
			}
			if !reflect.DeepEqual(tt.m, m) {
				t.Fatalf("got map %#v, want %#v", m, tt.m)
			}
		})
	}
}
