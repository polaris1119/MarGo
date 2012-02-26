package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
)

var (
	methods = map[string]Method{}
)

type Request map[string]string

type data interface{}

type Response struct {
	Error string `json:"error"`
	Data  data   `json:"data"`
}

type Method func(r Request) (data, error)

func respond(conn net.Conn) {
	resp := Response{}
	r := Request{}
	err := json.NewDecoder(conn).Decode(&r)
	if err == nil {
		meth := methods[r["call"]]
		if meth == nil {
			err = errors.New("Invalid method call `" + r["call"] + "`")
		} else {
			resp.Data, err = meth(r)
		}
	}
	if err != nil {
		resp.Error = err.Error()
	}
	err = json.NewEncoder(conn).Encode(resp)
	conn.Close()
}

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	addr := flag.String("addr", "127.9.5.1:57951", "The tcp address to listen on")
	flag.Parse()

	if *d {
		cmdpath, err := exec.LookPath(os.Args[0])
		if err != nil {
			log.Fatalln(err)
		}
		args := []string{os.Args[0], "-addr", *addr}
		attr := &os.ProcAttr{Files: []*os.File{nil, nil, nil}}
		p, err := os.StartProcess(cmdpath, args, attr)
		if err != nil {
			log.Fatalln(err)
		}
		p.Release()
	} else {
		l, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalln(err)
		}
		for {
			conn, err := l.Accept()
			if err != nil {
				if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
					log.Fatalln(err)
				}
			}
			go respond(conn)
		}
	}
}
