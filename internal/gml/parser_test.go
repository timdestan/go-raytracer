package gml

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseExamples(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  TokenList
	}{
		{
			name:  "empty",
			input: "",
			want:  tokens(),
		},
		{
			name:  "sphere",
			input: TestdataSphere,
			want: tokens(
				function(
					binder("v"), binder("u"), binder("face"),
					0.8, 0.2, sym("v"), sym("point"), 1.0, 0.2, 1.0,
				),
				sym("sphere"),
				binder("s"),
				sym("s"),
				-1.2,
				0.0,
				3.0,
				sym("translate"),
				sym("s"),
				1.2,
				1.0,
				3.0,
				sym("translate"),
				sym("union"),
				binder("scene"),
				-10.0, 10.0, 0.0, sym("point"),
				1.0, 1.0, 1.0, sym("point"),
				sym("pointlight"),
				binder("l"),
				0.5,
				0.5,
				0.5,
				sym("point"),
				array(sym("l")),
				sym("scene"),
				4,
				90.0,
				1920,
				1200,
				"sphere.ppm",
				sym("render"),
				// Trailing junk
				&Function{},
				&Array{},
				binder("ident"),
				true,
				false,
				123,
				1.23,
				"hello",
			),
		},
		{
			name:  "cube",
			input: TestdataCube,
			want: tokens(
				function(
					binder("v"), binder("u"), binder("face"),
					1.0, 0.5, 0.5, sym("point"),
					1.0, 0.0, 1.0,
				),
				sym("cube"),
				0.0, -0.5, 4.0, sym("translate"),
				2.0, sym("uscale"),
				45.0, sym("rotatex"),
				135.0, sym("rotatey"), binder("c"),
				1.0, 1.0, 1.0, sym("point"), binder("white"),
				0.0, 0.0, 1.0, sym("point"), binder("blue"),
				array(
					array(sym("blue"), sym("white")),
					array(sym("white"), sym("blue")),
				),
				binder("texture"),
				function(
					binder("i"),
					sym("i"), 0.0, sym("lessf"),
					function(sym("i"), sym("negf"), 0.5, sym("addf")),
					function(sym("i")), sym("if"),
				),
				binder("fabs"),
				function(
					sym("fabs"), sym("apply"), binder("v"),
					sym("fabs"), sym("apply"), binder("u"),
					binder("face"),
					function(
						sym("frac"), 0.5, sym("addf"), sym("floor"), binder("i"),
						sym("i"),
					),
					binder("toIntCoord"),
					sym("texture"), sym("u"), sym("toIntCoord"), sym("apply"), sym("get"),
					sym("v"), sym("toIntCoord"), sym("apply"), sym("get"),
					0.3, 0.9, 1.0,
				),
				sym("plane"),
				0.0, -3.0, 0.0, sym("translate"),
				binder("p"),
				function(
					binder("v"), binder("u"), binder("face"),
					0.5, 0.5, 0.5, sym("point"),
					0.3, 0.85, 1.0,
				),
				sym("plane"),
				0.0, 0.0, 8.0, sym("translate"),
				270.0, sym("rotatex"),
				45.0, sym("rotatez"),
				binder("p2"),

				sym("c"), sym("p"), sym("union"), sym("p2"), sym("union"), binder("scene"),
				-10, 10, 0, sym("point"),
				1.0, 1.0, 1.0, sym("point"), sym("pointlight"), binder("l"),
				0.2, 0.2, 0.2, sym("point"),
				array(sym("l")),
				sym("scene"),
				7,
				90.0,
				480, 320,
				"cube.ppm",
				sym("render"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			got, err := p.Parse()
			if err != nil {
				t.Errorf("Parse() error = %v", err)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Parse() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestParseScientificNotation(t *testing.T) {
	got, err := NewParser("1e3").Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if diff := cmp.Diff(got, tokens(1.0e3)); diff != "" {
		t.Errorf("Parse() mismatch (-got +want):\n%s", diff)
	}
}
