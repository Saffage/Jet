package parser

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/saffage/jet/ast"
	"github.com/saffage/jet/config"
	"github.com/saffage/jet/scanner"
	"github.com/saffage/jet/token"
)

var cfg *config.Config

// TODO more tests

func TestMain(m *testing.M) {
	cleanup()
	code := m.Run()
	os.Exit(code)
}

func TestMatchSequence(t *testing.T) {
	tokens := []token.Token{
		{Kind: token.At},
		{Kind: token.Ident},
		{Kind: token.At},
		{Kind: token.Ident},
		{Kind: token.At},
		{Kind: token.Ident},
	}
	p := New(nil, tokens, DefaultFlags)
	kinds := []token.Kind{token.At, token.Ident}
	if !p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return true, got false", kinds)
	}
	p.current += 2
	if !p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return true, got false", kinds)
	}
	p.current += 2
	if !p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return true, got false", kinds)
	}

	p.current = 0
	kinds = []token.Kind{token.At}
	if !p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return true, got false", kinds)
	}
	p.current += 2
	if !p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return true, got false", kinds)
	}
	p.current += 2
	if !p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return true, got false", kinds)
	}

	p.current = 0
	kinds = []token.Kind{token.At, token.Ident, token.Ident}
	if p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return false, got true", kinds)
	}
	p.current += 2
	if p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return false, got true", kinds)
	}
	p.current += 2
	if p.matchSequence(kinds...) {
		t.Errorf("expected `matchSequence(%v)` to return false, got true", kinds)
	}
}

func cleanup() {
	println("cleanup")
	cfg = config.New()
}

type testCase struct {
	input        string
	name         string
	expectedJSON string
	error        error
	isExpr       bool
	scannerFlags scanner.Flags
	parserFlags  Flags
}

func TestExprs(t *testing.T) {
	t.Cleanup(cleanup)

	cases := []testCase{
		{
			input:        `10`,
			name:         "untyped integer literal",
			expectedJSON: "untyped_integer_literal_ast.json",
			isExpr:       true,
		},
		{
			input:        `'hi'`,
			name:         "untyped string literal",
			expectedJSON: "untyped_string_literal_ast.json",
			isExpr:       true,
		},
		// TODO fix the test cases below
		// {
		// 	input:        `()`,
		// 	name:         "empty parentheses",
		// 	expectedJSON: "empty_parentheses.json",
		// 	isExpr:       true,
		// },
		// {
		// 	input:        `(,)`,
		// 	name:         "empty tuple constructor",
		// 	expectedJSON: "empty_tuple_constructor.json",
		// 	errors:       []error{errors.New("expected operand, found ','")},
		// 	isExpr:       true,
		// },
		// {
		// 	input:        `(1)`,
		// 	name:         "parentheses with 1 expr",
		// 	expectedJSON: "parentheses_with_1_expr.json",
		// 	isExpr:       true,
		// },
		// {
		// 	input:        `(1,)`,
		// 	name:         "tuple constructor",
		// 	expectedJSON: "tuple_constructor.json",
		// 	isExpr:       true,
		// },
	}

	for _, c := range cases {
		test(t, c)
	}
}

func test(t *testing.T, c testCase) {
	t.Run(c.name, func(t *testing.T) {
		tokens := scanner.MustScan(([]byte)(c.input), 1, c.scannerFlags)

		var stmts *ast.StmtList
		var err error

		if c.isExpr {
			var node ast.Node
			node, err = ParseExpr(cfg, tokens, c.parserFlags)
			stmts = &ast.StmtList{Nodes: []ast.Node{node}}
		} else {
			stmts, err = Parse(cfg, tokens, c.parserFlags)
		}
		if !checkError(t, err, c.error) {
			return
		}

		if c.expectedJSON != "" {
			filename := "./testdata/" + c.expectedJSON
			expect, err := os.ReadFile(filename)
			if err != nil {
				t.Errorf("unexpected error while reading file '%s': %s", filename, err)
				return
			}

			actual, err := json.MarshalIndent(stmts, "", "    ")
			if err != nil {
				t.Error("unexpected JSON marshal error:", err)
				return
			}

			if slices.Compare(actual, expect) != 0 {
				t.Errorf("invalid AST was parsed\nexpect %s\nactual %s", string(expect), string(actual))
				return
			}
		} else {
			encoded, err := json.MarshalIndent(stmts, "", "    ")
			if err != nil {
				t.Error("unexpected JSON marshal error:", err)
				return
			}

			t.Logf("no AST was expected\ngot %s", string(encoded))
		}
	})
}

func checkError(t *testing.T, got, want error) bool {
	if want == nil && got == nil {
		return true
	}

	if want == nil {
		if got != nil {
			t.Errorf("parsing failed with unexpected error: '%s'", got.Error())
			return false
		}
	} else if got == nil {
		t.Errorf("expected an error: '%s', got nothing", want.Error())
		return false
	}

	if got.Error() != want.Error() {
		t.Errorf(
			"unexpected error:\nexpect: '%s'\nactual: '%s'",
			got.Error(),
			want.Error(),
		)
		return false
	}

	return true
}
