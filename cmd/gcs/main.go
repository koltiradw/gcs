package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
)

var flagPathToMain = flag.String("path-to-main", "", "")

func main() {
	flag.Parse()
	if *flagPathToMain == "" {
		log.Fatalf("-path-to-main is required")
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, *flagPathToMain, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse file: %v", err)
	}

	ast.Inspect(node, func(d ast.Node) bool {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			startCoverageCall := &ast.GoStmt{
				Call: &ast.CallExpr{
					Fun: ast.NewIdent("StartCoverageServer"),
				},
			}
			fn.Body.List = append([]ast.Stmt{startCoverageCall}, fn.Body.List...)
			return false
		}
		return true
	})

	newImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Value: `"github.com/koltiradw/gcs"`,
		},
	}

	node.Imports = append(node.Imports, newImport)

	f, err := os.Create(*flagPathToMain)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := printer.Fprint(f, fset, node); err != nil {
		panic(err)
	}
}
