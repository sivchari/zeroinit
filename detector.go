package zeroinit

import (
	"go/ast"
	"go/constant"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// ZeroValueDetector defines the interface for detecting zero values
type ZeroValueDetector interface {
	IsZeroValue(expr ast.Expr) bool
}

// DefaultZeroValueDetector implements zero value detection for various types
type DefaultZeroValueDetector struct {
	pass     *analysis.Pass
	analyzer TypeAnalyzer
}

// NewZeroValueDetector creates a new zero value detector
func NewZeroValueDetector(pass *analysis.Pass, analyzer TypeAnalyzer) ZeroValueDetector {
	return &DefaultZeroValueDetector{
		pass:     pass,
		analyzer: analyzer,
	}
}

// IsZeroValue checks if an expression is a zero value
func (d *DefaultZeroValueDetector) IsZeroValue(expr ast.Expr) bool {
	// First try to use type information for accurate detection
	if d.isZeroValueByType(expr) {
		return true
	}

	// Check using AST-based detection for specific patterns
	switch e := expr.(type) {
	case *ast.BasicLit:
		return d.isBasicLitZero(e)
	case *ast.Ident:
		return d.isIdentZero(e)
	case *ast.CompositeLit:
		return d.isCompositeLitZero(e)
	case *ast.CallExpr:
		return d.isCallExprZero(e)
	}
	return false
}

func (d *DefaultZeroValueDetector) isZeroValueByType(expr ast.Expr) bool {
	tv, ok := d.pass.TypesInfo.Types[expr]
	if !ok {
		return false
	}

	// Check if it's a constant with zero value
	if tv.Value != nil {
		switch tv.Value.Kind() {
		case constant.Bool:
			return !constant.BoolVal(tv.Value)
		case constant.String:
			return constant.StringVal(tv.Value) == ""
		case constant.Int, constant.Float, constant.Complex:
			return constant.Sign(tv.Value) == 0
		}
	}

	return false
}

func (d *DefaultZeroValueDetector) isBasicLitZero(lit *ast.BasicLit) bool {
	switch lit.Kind {
	case token.INT:
		return lit.Value == "0"
	case token.FLOAT:
		return lit.Value == "0.0" || lit.Value == "0."
	case token.STRING:
		return lit.Value == `""`
	case token.CHAR:
		return lit.Value == "''"
	}
	return false
}

func (d *DefaultZeroValueDetector) isIdentZero(ident *ast.Ident) bool {
	return ident.Name == "nil" || ident.Name == "false"
}

func (d *DefaultZeroValueDetector) isCompositeLitZero(lit *ast.CompositeLit) bool {
	return len(lit.Elts) == 0
}

func (d *DefaultZeroValueDetector) isCallExprZero(call *ast.CallExpr) bool {
	fun, ok := call.Fun.(*ast.Ident)
	if !ok || fun.Name != "make" {
		return false
	}

	if len(call.Args) == 0 {
		return false
	}

	if !d.isMakeableZeroType(call.Args[0]) {
		return false
	}

	return d.hasMakeZeroArgs(call)
}

func (d *DefaultZeroValueDetector) isMakeableZeroType(typeExpr ast.Expr) bool {
	switch arg := typeExpr.(type) {
	case *ast.ArrayType:
		return arg.Len == nil // slice type
	case *ast.MapType:
		return true
	case *ast.ChanType:
		return false // channels should use make
	}
	return false
}

func (d *DefaultZeroValueDetector) hasMakeZeroArgs(call *ast.CallExpr) bool {
	if len(call.Args) == 1 {
		return true
	}

	if len(call.Args) >= 2 {
		if lit, ok := call.Args[1].(*ast.BasicLit); ok {
			return lit.Kind == token.INT && lit.Value == "0"
		}
	}
	return false
}
