package gml

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var flagUpdate = flag.Bool("update", false, "If true, update the expected renderarg outputs")

// TestSimpleEval tests some simple cases with no render call.
func TestSimpleEval(t *testing.T) {
	type testCase struct {
		name    string
		program string
		want    Value // expected top of stack
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := NewParser(tt.program).Parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
				return
			}
			st := NewEvalState()
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

// TestSingleRender tests programs where we expect exactly one call to render.
func TestSingleRender(t *testing.T) {
	type testCase struct {
		name  string
		debug bool // set to enable debug tracing
	}
	for _, tt := range []testCase{
		{name: "sphere"},
		{name: "cube"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			program := MustReadTestdataFile("testdata/" + tt.name + ".gml")
			tokens, err := NewParser(program).Parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
				return
			}
			var got *RenderArgs
			st := NewEvalState()
			st.Render = func(e *EvalState, args *RenderArgs) error {
				if got == nil {
					got = args
				} else {
					t.Errorf("multiple render calls: %v", args)
				}
				return nil
			}
			st.Debug = tt.debug
			err = st.Eval(tokens)
			if err != nil {
				t.Errorf("eval error: %v", err)
				return
			}
			gotLines := RenderArgsToLines(got)
			wantFile := "testdata/" + tt.name + ".out"
			wantLines := strings.Split(MustReadTestdataFile(wantFile), "\n")
			if diff := cmp.Diff(wantLines, gotLines); diff != "" {
				if *flagUpdate {
					err := os.WriteFile(wantFile, []byte(strings.Join(gotLines, "\n")), 0644)
					if err != nil {
						t.Errorf("write error: %v", err)
					}
				} else {
					t.Errorf("Eval() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// Run benchmarks with:
// go test -run ^$ -bench . -cpuprofile=/tmp/cpu.prof
// go tool pprof -http=:8080 /tmp/cpu.prof

func BenchmarkParseAndEval(b *testing.B) {
	for b.Loop() {
		tokens, err := NewParser(MustReadTestdataFile("testdata/sphere.gml")).Parse()
		if err != nil {
			b.Errorf("parse error: %v", err)
			return
		}
		st := NewEvalState()
		st.Render = func(e *EvalState, args *RenderArgs) error {
			return nil
		}
		err = st.Eval(tokens)
		if err != nil {
			b.Errorf("eval error: %v", err)
			return
		}
	}
}

// Run benchmarks with:
// go test -run ^$ -bench . -cpuprofile=/tmp/cpu.prof
// go tool pprof -http=:8080 /tmp/cpu.prof

func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		_, err := NewParser(MustReadTestdataFile("testdata/sphere.gml")).Parse()
		if err != nil {
			b.Errorf("parse error: %v", err)
			return
		}
	}
}

func BenchmarkEval(b *testing.B) {
	tokens, err := NewParser(MustReadTestdataFile("testdata/sphere.gml")).Parse()
	if err != nil {
		b.Errorf("parse error: %v", err)
		return
	}
	for b.Loop() {
		st := NewEvalState()
		st.Render = func(e *EvalState, args *RenderArgs) error {
			return nil
		}
		err = st.Eval(tokens)
		if err != nil {
			b.Errorf("eval error: %v", err)
			return
		}
	}
}
