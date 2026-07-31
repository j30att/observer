package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = "checks for panic and process termination calls outside main"

var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  doc,
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if isGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				inspectNode(pass, decl, false)
				continue
			}

			inspectNode(pass, fn.Body, fn.Name.Name == "main" && pass.Pkg.Name() == "main")
		}
	}

	return nil, nil
}

func inspectNode(pass *analysis.Pass, node ast.Node, inMain bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		switch call := n.(type) {
		case *ast.FuncDecl:
			return false
		case *ast.CallExpr:
			switch {
			case isBuiltinPanic(pass, call):
				pass.Reportf(call.Pos(), "usage of panic is prohibited")
			case !inMain && isLogFatal(pass, call):
				pass.Reportf(call.Pos(), "log.Fatal call is allowed only in main function of main package")
			case !inMain && isOSExit(pass, call):
				pass.Reportf(call.Pos(), "os.Exit call is allowed only in main function of main package")
			}
		}

		return true
	})
}

func isGenerated(file *ast.File) bool {
	if file.Pos() == token.NoPos {
		return false
	}

	return ast.IsGenerated(file)
}

func isBuiltinPanic(pass *analysis.Pass, call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "panic" {
		return false
	}

	return pass.TypesInfo.Uses[ident] == types.Universe.Lookup("panic")
}

func isLogFatal(pass *analysis.Pass, call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !strings.HasPrefix(selector.Sel.Name, "Fatal") {
		return false
	}

	return isPackageSelector(pass, selector, "log")
}

func isOSExit(pass *analysis.Pass, call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Exit" {
		return false
	}

	return isPackageSelector(pass, selector, "os")
}

func isPackageSelector(pass *analysis.Pass, selector *ast.SelectorExpr, importPath string) bool {
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
	return ok && pkgName.Imported().Path() == importPath
}
