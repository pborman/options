package options

import (
	"bytes"
	"fmt"
	"strings"
)

// unescape returns line with leading/trailing spaces and comments stripped as
// well as backslash processing have been done.
func unescape(line []byte) string {
	line = bytes.TrimLeft(line, " \t")
	if len(line) == 0 || line[0] == '#' {
		return ""
	}
	escape := false
	p := 0
Loop:
	for _, c := range line {
		switch {
		case escape:
			escape = false
		case c == '\\':
			escape = true
			continue
		case c == '#':
			break Loop
		}
		line[p] = c
		p++
	}
	return string(bytes.TrimSpace(line[:p]))
}

// SimpleDecoder decodes data as a set of name=value pairs, one pair per line.
// Keys and values are separated by an equals sign (=), with optional white
// space on either side of the equal sign.  Comments are introduced by the pound
// (#) character, unless prefaced by a backslash (\).  \X is replaced with X.  A
// backslash at the end of the line is ignored (no line concatination).  If the
// value begins and ends with double quote ("), the double duotes are trimmed
// (but no futher processing is done).  A non-backslashed # within quotes still
// introduces a comment.
//
// Examples lines:
//
//	# this is a comment
//	name=value
//	name=a value
//	name = a value  # with a comment
//	name= "a value"
//	name = \# is the value # this is the comment
//	name = " a value with spaces "
//	set.name = value # set name in Options set "name"
func SimpleDecoder(data []byte) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	for n, d := range bytes.Split(data, []byte{'\n'}) {
		line := unescape(d)
		if line == "" {
			continue
		}
		x := strings.Index(line, "=")
		if x < 0 {
			return nil, fmt.Errorf("line %d: missing value: %q", n+1, line)
		}
		if x == 0 {
			return nil, fmt.Errorf("line %d: missing name: %q", n+1, line)
		}
		name := strings.TrimSpace(line[:x])
		if strings.Index(name, " ") >= 0 {
			return nil, fmt.Errorf("line %d: space in name: %q", n+1, line)
		}
		value := strings.TrimSpace(line[x+1:])
		if e := len(value); e > 1 && value[0] == '"' && value[e-1] == '"' {
			value = value[1 : e-1]
		}
		fields := strings.Split(name, ".")
		m := m
		for len(fields) > 1 {
			switch m1 := m[fields[0]].(type) {
			case nil:
				nm := map[string]interface{}{}
				m[fields[0]] = nm
				m = nm
			case map[string]interface{}:
				m = m1
			default:
				return nil, fmt.Errorf("%s: conflict on field %s", name, fields[0])
			}
			fields = fields[1:]
		}
		m[fields[0]] = value
	}
	return m, nil
}
