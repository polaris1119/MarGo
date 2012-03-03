package main

import (
	"go/ast"
	"go/parser"
	"go/token"
)

type PackageNameArgs struct {
	Fn  string `json:"fn"`
	Src string `json"src"`
}

func init() {
	methods["package_name"] = func(r Request) (data, error) {
		a := PackageNameArgs{}
		if err := r.Decode(&a); err != nil {
			return "", err
		}

		package_name := ""
		fset := token.NewFileSet()
		var err error
		var src interface{}
		if a.Src != "" {
			src = a.Src
		}
		if err == nil {
			if a.Fn == "" {
				a.Fn = "<stdin>"
			}
			var af *ast.File
			af, err = parser.ParseFile(fset, a.Fn, src, parser.PackageClauseOnly)
			if af != nil {
				package_name = af.Name.String()
			}
		}
		return package_name, err
	}
}
