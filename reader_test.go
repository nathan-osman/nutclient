package nutclient

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

func TestSplit(t *testing.T) {
	for _, v := range []struct {
		name   string
		input  string
		output []string
		err    bool
	}{
		{
			name:   "empty input",
			input:  "",
			output: []string{},
		},
		{
			name:   "whitespace",
			input:  " test\ttest\n",
			output: []string{"test", "test"},
		},
		{
			name:   "string",
			input:  "a \"b c\" d",
			output: []string{"a", "b c", "d"},
		},
		{
			name:   "string (error)",
			input:  "a \"b",
			output: []string{"a"},
			err:    true,
		},
	} {
		s := bufio.NewScanner(strings.NewReader(v.input))
		s.Split(split)
		output := []string{}
		for s.Scan() {
			t := s.Text()
			if len(t) != 0 {
				output = append(output, t)
			}
		}
		if !reflect.DeepEqual(v.output, output) {
			t.Fatalf("%s: %#v != %#v", v.name, v.output, output)
		}
		if v.err != (s.Err() != nil) {
			t.Fatalf("%s: %#v != %#v", v.name, v.err, s.Err())
		}
	}
}

func TestListReader(t *testing.T) {
	for _, v := range []struct {
		name   string
		input  string
		output map[string]string
		err    bool
	}{
		{
			name:  "empty input",
			input: "",
			err:   true,
		},
		{
			name: "empty list",
			input: `BEGIN LIST ups
END LIST VAR ups`,
			output: map[string]string{},
		},
		{
			name: "two variables",
			input: `BEGIN LIST ups
VAR ups k1 "v1"
VAR ups k2 "v2"
END LIST VAR ups`,
			output: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
		},
	} {
		l, err := newListReader(strings.NewReader(v.input))
		if err != nil {
			if !v.err {
				t.Fatalf("%s: %s", v.name, err)
			}
		} else {
			if !reflect.DeepEqual(v.output, l.variables) {
				t.Fatalf("%s: %#v != %#v", v.name, v.output, l.variables)
			}
		}
	}
}
