package gml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEval(t *testing.T) {
	type testCase struct {
		name    string
		program string
		want    Value
		debug   bool // set to enable debug tracing
	}
	for _, tt := range []testCase{
		{
			name:    "apply",
			program: "1 { /x x x } apply addi",
			want:    VInt(2),
		},
		{
			name: "rebind",
			program: `
					1 /x           % bind x to 1
					{ x } /f       % the function f pushes the value of x
					2 /x           % rebind x to 2
					f apply x addi`,
			want: VInt(3),
		},
		// {
		// 	name:    "sphere",
		// 	program: TestdataSphere,
		// },
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := NewParser(tt.program).Parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
				return
			}
			st := NewEvalState()
			if tt.debug {
				st.Tracer = func(s string) {
					fmt.Println(s)
				}
				printInBox(tt.name)
			}
			err = st.Eval(tokens)
			if err != nil {
				t.Errorf("eval error: %v", err)
				return
			}
			var got Value
			if len(st.Stack) > 0 {
				// Check the last value on the stack.
				got = st.Stack[len(st.Stack)-1]
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Eval() mismatch (-got +want):\n%s", diff)
			}
		})
	}

}

func printInBox(msg string) {
	msg = fmt.Sprintf("**  %s  **", msg)
	rowOfStars := strings.Repeat("*", len(msg))
	fmt.Printf("%s\n%s\n%s\n", rowOfStars, msg, rowOfStars)
}
