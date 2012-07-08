package main

import (
	"os"
	"path/filepath"
	"strings"
)

type PkgDirsArgs struct {
	Env map[string]string `json:"env"`
}

func init() {
	act(Action{
		Path: "/pkgdirs",
		Doc:  ``,
		Func: func(r Request) (data, error) {
			a := PkgDirsArgs{
				Env: map[string]string{},
			}
			if err := r.Decode(&a); err != nil {
				return map[string]map[string]string{}, err
			}
			return pkgDirs(a.Env), nil
		},
	})
}

func pkgDirs(env map[string]string) map[string]map[string]string {
	res := map[string]map[string]string{}
	for _, root := range rootDirs(env) {
		res[root] = map[string]string{}
		filepath.Walk(root, func(fn string, fi os.FileInfo, err error) error {
			if err == nil {
				name := fi.Name()
				if name[0] == '.' || name[0] == '_' {
					if fi.IsDir() {
						return filepath.SkipDir
					}
				} else if !fi.IsDir() && strings.HasSuffix(fn, ".go") {
					if dir, err := filepath.Rel(root, filepath.Dir(fn)); err == nil {
						dir = filepath.ToSlash(dir)
						if _, ok := res[root][dir]; !ok {
							res[root][dir] = fn
						}
					}
				}
			}
			return nil
		})
	}
	return res
}
