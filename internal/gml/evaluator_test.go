package gml

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func evalError(t *testing.T, program string) error {
	t.Helper()
	err := NewEvalState().ParseAndEval(program)
	if err == nil {
		t.Fatal("expected an eval error, got nil")
	}
	return err
}

var flagUpdate = flag.Bool("update", false, "If true, update the expected renderarg outputs")

// TestEvalErrorPositions verifies that evaluator errors include the source position
// of the token that caused the failure.
func TestEvalErrorPositions(t *testing.T) {
	tests := []struct {
		name       string
		program    string
		wantPrefix string
	}{
		{
			// addi is on line 3; popping VReal where VInt is expected
			name:       "type mismatch on line 3",
			program:    "1\n2.0\naddi",
			wantPrefix: "3:1:",
		},
		{
			// unbound identifier is on line 2, col 3
			name:       "unbound identifier",
			program:    "1 /x\n1 missing",
			wantPrefix: "2:3:",
		},
		{
			// empty stack: the lonely addi has nothing to pop
			name:       "empty stack",
			program:    "addi",
			wantPrefix: "1:1:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := evalError(t, tt.program)
			if !strings.HasPrefix(err.Error(), tt.wantPrefix) {
				t.Errorf("error %q does not start with %q", err.Error(), tt.wantPrefix)
			}
		})
	}
}

// TestSimpleEval tests some simple cases with no render call.
func TestSimpleEval(t *testing.T) {
	type testCase struct {
		name    string
		program string
		want    Value // expected top of stack
		debug   bool
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
		{
			name:    "frac",
			program: `3.75 frac`,
			want:    VReal(0.75),
		},
		{
			name:    "frac (negative)",
			program: `-3.75 frac`,
			want:    VReal(-0.75),
		},
		{
			name: "if (true)",
			program: `
				-1.0 /i
				i 0.0 lessf { i negf 0.5 addf } { i } if`,
			want: VReal(1.5),
		},
		{
			name: "if (false)",
			program: `
				2.5 /i
				i 0.0 lessf { i negf 0.5 addf } { i } if`,
			want: VReal(2.5),
		},
		{
			name: "env preserved",
			program: `
				1 /i
				{ i /j /i i j addi } /f
				2 f apply  % stack: 3
				f apply    % stack: 4 
				`,
			want: VInt(4),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			st := NewEvalState()
			st.Debug = tt.debug
			err := st.ParseAndEval(tt.program)
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
				t.Errorf("Eval(%s) mismatch (-got +want):\n%s", tt.name, diff)
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
			err := st.ParseAndEval(program)
			if err != nil {
				t.Errorf("eval error: %v", err)
				return
			}
			gotLines := RenderArgsToLines(got, &st.IDMapping)
			wantFile := "testdata/" + tt.name + ".out"
			wantLines := SplitLines(MustReadTestdataFile(wantFile))
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
	program := MustReadTestdataFile("testdata/sphere.gml")
	for b.Loop() {
		st := NewEvalState()
		st.Render = func(e *EvalState, args *RenderArgs) error {
			return nil
		}
		err := st.ParseAndEval(program)
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
	program := MustReadTestdataFile("testdata/sphere.gml")
	for b.Loop() {
		_, err := NewParser(program).Parse()
		if err != nil {
			b.Errorf("parse error: %v", err)
			return
		}
	}
}
