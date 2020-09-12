package curly_test

import (
	"strings"
	"testing"

	"github.com/skillian/curly"
)

type curlyTest struct {
	name   string
	format string
	value  interface{}
	expect string
	err    string
}

type Hello struct {
	Greeting string
	Whom     string
}

var curlyTests = []curlyTest{
	curlyTest{
		name:   "helloWorld",
		format: "Hello, {Whom}",
		value:  Hello{Whom: "World"},
		expect: "Hello, World",
		err:    "",
	},
	curlyTest{
		name:   "greetingWorld",
		format: "{Greeting}, {Whom}, how are you?",
		value:  Hello{Greeting: "Hello", Whom: "World"},
		expect: "Hello, World, how are you?",
		err:    "",
	},
}

func TestCurly(t *testing.T) {
	for _, tc := range curlyTests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s, err := curly.Format(tc.format, tc.value)
			if err != nil {
				if tc.err != "" {
					if strings.Contains(err.Error(), tc.err) {
						return
					}
				}
				t.Fatal(err)
			}
			if s != tc.expect {
				t.Fatalf("%q != %q", s, tc.expect)
			}
		})
	}
}
