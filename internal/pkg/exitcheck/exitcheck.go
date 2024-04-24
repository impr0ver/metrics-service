// Exitcheck include analysis.Analyzer, that find os.Exit in main function of main packet.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// analysis.Analyzer, check direct os.Exit in function main.main().
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check for direct call os.Exit in main.main()",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// ast.Inspect function walk for AST nodes
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			// skip not packet-main nodes
			case *ast.Package:
				if x.Name != "main" {
					return false
				}
			// skip not main-functions nodes
			case *ast.FuncDecl:
				if x.Name.Name != "main" {
					return false
				}
				for _, stmt := range x.Body.List {
					ast.Inspect(stmt, func(node ast.Node) bool {
						switch x := node.(type) {
						case *ast.CallExpr:
							f, ok := x.Fun.(*ast.SelectorExpr)
							if !ok {
								break
							}
							fx, ok := f.X.(*ast.Ident)
							if !ok {
								break
							}
							if fx.Name == "os" && f.Sel.Name == "Exit" {
								pass.Reportf(f.Pos(), "call os.Exit in main.main function is not use!")
							}
						}
						return true
					})
				}
				return false
			}
			return true
		})
	}
	return nil, nil
}