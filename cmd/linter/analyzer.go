package main

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
)

// Analyzer — основной анализатор, проверяющий использование panic, log.Fatal и os.Exit
var Analyzer = &analysis.Analyzer{
	Name: "linter",
	Doc:  "проверяет использование panic, log.Fatal и os.Exit вне main.main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CallExpr:
				checkCallExpr(pass, n)
			}
			return true
		})
	}
	return nil, nil
}

func checkCallExpr(pass *analysis.Pass, call *ast.CallExpr) {
	// Проверяем вызов panic
	if isPanicCall(call, pass) {
		pass.Reportf(call.Fun.Pos(), "использование panic не рекомендуется")
		return
	}

	// Проверяем вызовы log.Fatal и os.Exit
	if isLogFatalOrOsExitCall(call, pass) {
		// Проверяем, что это не функция main в пакете main
		if !isMainMainFunction(pass, call) {
			funcName := getCalledFunctionName(call)
			pass.Reportf(call.Fun.Pos(), "вызов %s вне main.main не рекомендуется", funcName)
		}
	}
}

func isPanicCall(call *ast.CallExpr, pass *analysis.Pass) bool {
	fun, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	return fun.Name == "panic"
}

func isLogFatalOrOsExitCall(call *ast.CallExpr, pass *analysis.Pass) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	pkgName := ident.Name
	funcName := selector.Sel.Name

	return (pkgName == "log" && funcName == "Fatal") ||
		(pkgName == "os" && funcName == "Exit")
}

func isMainMainFunction(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Получаем текущее положение вызова
	pos := call.Pos()
	file := pass.Fset.File(pos)
	line := file.Line(pos)

	// Ищем функцию main в пакете main
	for _, file := range pass.Files {
		if file.Name.Name != "main" {
			continue
		}

		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Name.Name != "main" {
				continue
			}

			// Проверяем, находится ли вызов внутри функции main
			startLine := pass.Fset.Position(funcDecl.Body.Pos()).Line
			endLine := pass.Fset.Position(funcDecl.Body.End()).Line

			if line >= startLine && line <= endLine {
				return true
			}
		}
	}
	return false
}

func getCalledFunctionName(call *ast.CallExpr) string {
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		return selector.Sel.Name
	}
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return ident.Name
	}
	return "unknown"
}
