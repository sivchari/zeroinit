package zeroinit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "zeroinit checks for zero value initialization using short variable declaration and suggests using var declaration instead"

// Analyzer checks for zero value initialization using short variable declaration
var Analyzer = &analysis.Analyzer{
	Name: "zeroinit",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		assignStmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return
		}

		// Check if it's a short variable declaration (:=)
		if assignStmt.Tok != token.DEFINE {
			return
		}

		// Check each assignment
		for i, rhs := range assignStmt.Rhs {
			if i >= len(assignStmt.Lhs) {
				continue
			}

			lhs, ok := assignStmt.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}

			// Check if RHS is a zero value
			if isZeroValue(pass, rhs) {
				// Get the type of the variable
				varType := getTypeString(pass, rhs)
				if varType == "" {
					continue
				}

				// Create suggested fix
				fix := createSuggestedFix(pass, assignStmt, lhs, varType)

				pass.Report(analysis.Diagnostic{
					Pos:     assignStmt.Pos(),
					Message: fmt.Sprintf("should use var declaration for zero value of %s", varType),
					SuggestedFixes: []analysis.SuggestedFix{
						fix,
					},
				})
			}
		}
	})

	return nil, nil
}

func isZeroValue(pass *analysis.Pass, expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return e.Value == "0"
		case token.FLOAT:
			return e.Value == "0.0" || e.Value == "0."
		case token.STRING:
			return e.Value == `""`
		case token.CHAR:
			return e.Value == "''"
		}
	case *ast.Ident:
		// Check for nil, false
		return e.Name == "nil" || e.Name == "false"
	case *ast.CompositeLit:
		// Empty composite literal (e.g., []int{}, map[string]int{})
		return len(e.Elts) == 0
	case *ast.CallExpr:
		// Check for make with zero length/capacity
		if fun, ok := e.Fun.(*ast.Ident); ok && fun.Name == "make" {
			if len(e.Args) == 0 {
				return false
			}

			// Check the type being made
			var isSliceOrMap bool
			switch arg := e.Args[0].(type) {
			case *ast.ArrayType:
				// slice type: []T
				isSliceOrMap = arg.Len == nil
			case *ast.MapType:
				// map type: map[K]V
				isSliceOrMap = true
			case *ast.ChanType:
				// channel: make(chan T) without buffer is NOT a zero value
				// make(chan T, 0) with explicit 0 buffer is also NOT a zero value
				// channels should be initialized with make, not var
				return false
			}

			if !isSliceOrMap {
				return false
			}

			// For slices and maps:
			// make([]T) or make(map[K]V) without size arguments
			if len(e.Args) == 1 {
				return true
			}
			// make([]T, 0) or make([]T, 0, 0)
			if len(e.Args) >= 2 {
				if lit, ok := e.Args[1].(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "0" {
					return true
				}
			}
		}
	}
	return false
}

func getTypeString(pass *analysis.Pass, expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float64"
		case token.STRING:
			return "string"
		case token.CHAR:
			return "rune"
		}
	case *ast.Ident:
		if e.Name == "nil" {
			// Cannot determine type for nil without more context
			return ""
		}
		if e.Name == "false" {
			return "bool"
		}
	case *ast.CompositeLit:
		if e.Type != nil {
			var buf bytes.Buffer
			if err := format.Node(&buf, pass.Fset, e.Type); err == nil {
				return buf.String()
			}
		}
	case *ast.CallExpr:
		if fun, ok := e.Fun.(*ast.Ident); ok && fun.Name == "make" {
			if len(e.Args) > 0 {
				var buf bytes.Buffer
				if err := format.Node(&buf, pass.Fset, e.Args[0]); err == nil {
					return buf.String()
				}
			}
		}
	}

	// Try to get type from type checker
	if tv, ok := pass.TypesInfo.Types[expr]; ok {
		return tv.Type.String()
	}

	return ""
}

func createSuggestedFix(pass *analysis.Pass, assignStmt *ast.AssignStmt, ident *ast.Ident, varType string) analysis.SuggestedFix {
	// Create var declaration
	varDecl := &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{ident},
					Type:  createTypeExpr(varType),
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, pass.Fset, varDecl); err != nil {
		return analysis.SuggestedFix{}
	}

	return analysis.SuggestedFix{
		Message: fmt.Sprintf("Replace with var %s %s", ident.Name, varType),
		TextEdits: []analysis.TextEdit{
			{
				Pos:     assignStmt.Pos(),
				End:     assignStmt.End(),
				NewText: buf.Bytes(),
			},
		},
	}
}

func createTypeExpr(typeStr string) ast.Expr {
	// Simple type expression creation for common types
	// In a real implementation, you'd want to parse the type string properly
	return &ast.Ident{Name: typeStr}
}
