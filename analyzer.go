package zeroinit

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// TypeAnalyzer provides type information about expressions
type TypeAnalyzer interface {
	GetExprType(expr ast.Expr) (types.Type, bool)
	IsUntyped(expr ast.Expr) bool
}

// DefaultTypeAnalyzer provides type information from TypesInfo
type DefaultTypeAnalyzer struct {
	pass *analysis.Pass
}

// NewTypeAnalyzer creates a new type analyzer
func NewTypeAnalyzer(pass *analysis.Pass) TypeAnalyzer {
	return &DefaultTypeAnalyzer{pass: pass}
}

// GetExprType returns the type of an expression
func (a *DefaultTypeAnalyzer) GetExprType(expr ast.Expr) (types.Type, bool) {
	if tv, ok := a.pass.TypesInfo.Types[expr]; ok {
		return tv.Type, true
	}
	return nil, false
}

// IsUntyped checks if an expression has an untyped type
func (a *DefaultTypeAnalyzer) IsUntyped(expr ast.Expr) bool {
	if tv, ok := a.pass.TypesInfo.Types[expr]; ok {
		if basic, ok := tv.Type.(*types.Basic); ok {
			return basic.Info()&types.IsUntyped != 0
		}
	}
	return false
}
