package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
)

type PkgFilesArgs struct {
	Path string `json:"path"`
}

func init() {
	act(Action{
		Path: "/pkgfiles",
		Doc:  ``,
		Func: func(r Request) (data, error) {
			res := map[string][]string{}
			a := PkgFilesArgs{}
			if err := r.Decode(&a); err != nil {
				return res, err
			}

			srcDir, err := filepath.Abs(a.Path)
			if err != nil {
				return res, err
			}

			fset := token.NewFileSet()
			pkgs, _ := parser.ParseDir(fset, srcDir, nil, parser.PackageClauseOnly)
			if pkgs != nil {
				for pkgName, pkg := range pkgs {
					list := []string{}
					for _, f := range pkg.Files {
						tp := fset.Position(f.Pos())
						fn, _ := filepath.Rel(srcDir, tp.Filename)
						if fn != "" {
							list = append(list, fn)
						}
					}
					if len(list) > 0 {
						res[pkgName] = list
					}
				}
			}

			return res, nil
		},
	})
}
