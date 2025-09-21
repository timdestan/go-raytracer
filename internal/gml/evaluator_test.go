package gml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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

var ignoreSurfaceFn = cmpopts.IgnoreFields(Sphere{}, "SurfaceFn")

// TestSingleRender tests programs where we expect exactly one call to render.
func TestSingleRender(t *testing.T) {
	type testCase struct {
		name           string
		program        string
		wantRenderArgs *RenderArgs
		debug          bool // set to enable debug tracing
	}
	for _, tt := range []testCase{
		{
			name:    "sphere",
			program: TestdataSphere,
			wantRenderArgs: &RenderArgs{
				AmbientLight: &Point{X: 0.5, Y: 0.5, Z: 0.5},
				Lights: []*PointLight{
					{
						Position: Point{X: -10.0, Y: 10.0, Z: 0.0},
						Color:    Point{X: 1.0, Y: 1.0, Z: 1.0},
					},
				},
				Scene: &Union{
					Objects: []SceneObject{
						&Sphere{
							Center: Point{X: 1.2, Y: 1.0, Z: 3.0},
							Radius: 1.0,
						},
						&Sphere{
							Center: Point{X: -1.2, Y: 0.0, Z: 3.0},
							Radius: 1.0,
						},
					},
				},
				Depth:  4,
				Fov:    90.0,
				Width:  1920,
				Height: 1200,
				File:   "sphere.ppm",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := NewParser(tt.program).Parse()
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
			if tt.debug {
				st.Tracer = func(s string) {
					fmt.Print(s)
				}
				printInBox(tt.name)
			}
			err = st.Eval(tokens)
			if err != nil {
				t.Errorf("eval error: %v", err)
				return
			}

			if diff := cmp.Diff(got, tt.wantRenderArgs, ignoreSurfaceFn); diff != "" {
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

// Run benchmarks with:
// go test -run ^$ -bench . -cpuprofile=/tmp/cpu.prof
// go tool pprof -http=:8080 /tmp/cpu.prof

func BenchmarkParseAndEval(b *testing.B) {
	for b.Loop() {
		tokens, err := NewParser(TestdataSphere).Parse()
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
		_, err := NewParser(TestdataSphere).Parse()
		if err != nil {
			b.Errorf("parse error: %v", err)
			return
		}
	}
}

func BenchmarkEval(b *testing.B) {
	tokens, err := NewParser(TestdataSphere).Parse()
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
