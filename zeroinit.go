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
	checker := NewZeroInitChecker(pass)
	return checker.Check()
}

// ZeroInitChecker coordinates the zero initialization checking process
type ZeroInitChecker struct {
	pass     *analysis.Pass
	detector ZeroValueDetector
	analyzer TypeAnalyzer
}

// NewZeroInitChecker creates a new checker instance
func NewZeroInitChecker(pass *analysis.Pass) *ZeroInitChecker {
	analyzer := NewTypeAnalyzer(pass)
	return &ZeroInitChecker{
		pass:     pass,
		detector: NewZeroValueDetector(pass, analyzer),
		analyzer: analyzer,
	}
}

// Check performs the zero initialization check
func (c *ZeroInitChecker) Check() (any, error) {
	inspect := c.pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	inspect.Preorder(nodeFilter, c.processNode)

	return nil, nil
}

func (c *ZeroInitChecker) processNode(n ast.Node) {
	assignStmt, ok := n.(*ast.AssignStmt)
	if !ok || assignStmt.Tok != token.DEFINE {
		return
	}

	c.checkAssignments(assignStmt)
}

func (c *ZeroInitChecker) checkAssignments(assignStmt *ast.AssignStmt) {
	for i, rhs := range assignStmt.Rhs {
		if i >= len(assignStmt.Lhs) {
			continue
		}

		lhs, ok := assignStmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}

		if c.detector.IsZeroValue(rhs) {
			c.reportZeroInit(assignStmt, lhs, rhs)
		}
	}
}

func (c *ZeroInitChecker) reportZeroInit(assignStmt *ast.AssignStmt, lhs *ast.Ident, rhs ast.Expr) {
	varType := c.getTypeString(rhs)
	if varType == "" {
		return
	}

	fix := c.createSuggestedFix(assignStmt, lhs, varType)

	c.pass.Report(analysis.Diagnostic{
		Pos:            assignStmt.Pos(),
		Message:        fmt.Sprintf("should use var declaration for zero value of %s", varType),
		SuggestedFixes: []analysis.SuggestedFix{fix},
	})
}

// createSuggestedFix creates a suggested fix for replacing := with var
func (c *ZeroInitChecker) createSuggestedFix(assignStmt *ast.AssignStmt, ident *ast.Ident, varType string) analysis.SuggestedFix {
	varDecl := &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{ident},
					Type:  &ast.Ident{Name: varType},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, c.pass.Fset, varDecl); err != nil {
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

// getTypeString returns a string representation of the expression's type
func (c *ZeroInitChecker) getTypeString(expr ast.Expr) string {
	if tv, ok := c.pass.TypesInfo.Types[expr]; ok && tv.Type != nil {
		return tv.Type.String()
	}
	return ""
}
