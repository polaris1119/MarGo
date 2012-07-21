package main

import (
	"os"
	"path"
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
		walkRootDir(root, res[root], root)
	}
	return res
}

func walkRootDir(root string, m map[string]string, basePath string) {
	dir, err := os.Open(root)
	if err != nil {
		return
	}

	importPath, err := filepath.Rel(basePath, root)
	if err != nil {
		importPath = root
	}
	importPath = path.Clean(filepath.ToSlash(importPath))
	idealName := path.Base(importPath) + ".go"

	names, _ := dir.Readdirnames(-1)
	for _, name := range names {
		if name[0] == '.' || name[0] == '_' {
			continue
		}

		fn := filepath.Join(root, name)
		if strings.HasSuffix(name, ".go") {
			isIdeal := false
			oldFn, ok := m[importPath]
			if ok {
				isIdeal = strings.HasSuffix(oldFn, idealName)
			}
			if !ok || name == idealName || (!isIdeal && name == "main.go") {
				m[importPath] = fn
			}
		} else if fi, err := os.Stat(fn); err == nil && fi.IsDir() {
			walkRootDir(fn, m, basePath)
		}
	}
}
