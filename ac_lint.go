package main

import (
	"go/parser"
	"go/scanner"
)

type AcLintArgs struct {
	Fn	string	`json:"fn"`
	Src	string	`json:"src"`
}

type AcLintReport struct {
	Row	int	`json:"row"`
	Col	int	`json:"col"`
	Msg	string	`json:"msg"`
}

func init() {
	act(Action{
		Path:	"/lint",
		Doc:	``,
		Func: func(r Request) (data, error) {
			a := AcLintArgs{}
			res := make([]AcLintReport, 0)

			if err := r.Decode(&a); err != nil {
				return res, err
			}

			_, _, err := parseAstFile(a.Fn, a.Src, parser.DeclarationErrors)
			if err != nil {
				if el, ok := err.(scanner.ErrorList); ok {
					for _, e := range el {
						res = append(res, AcLintReport{
							Row:	e.Pos.Line - 1,
							Col:	e.Pos.Column - 1,
							Msg:	e.Msg,
						})
					}
				}
			}

			return res, nil
		},
	})
}
