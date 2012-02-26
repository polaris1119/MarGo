package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
)

var (
	methods = map[string]Method{}
)

type Request struct {
	Method string            `json:"method"`
	Env    map[string]string `json:"env"`
	Args   Args              `json:"args"`
}

type Args struct {
	Filename string `json:"filename"`
	Src      string `json:"src"`
}

type data interface{}

type Response struct {
	Error string `json:"error"`
	Data  data   `json:"data"`
}

type Method func(r *Request) (data, error)

func respond(conn net.Conn) {
	resp := Response{}
	r := &Request{
		Env: map[string]string{},
	}
	err := json.NewDecoder(conn).Decode(&r)
	if err == nil {

		if r.Method != "exit" {

			meth := methods[r.Method]
			if meth == nil {
				err = errors.New("Invalid method call `" + r.Method + "`")
			} else {
				resp.Data, err = meth(r)
			}
		}
	}

	if err != nil {
		resp.Error = err.Error()
	}
	err = json.NewEncoder(conn).Encode(resp)
	conn.Close()

	if r.Method == "exit" {
		os.Exit(0)
	}
}

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	closeFds := flag.Bool("close-fds", false, "Whether or not to close stdin, stdout and stderr")
	addr := flag.String("addr", "127.9.5.1:57951", "The tcp address to listen on")
	flag.Parse()

	if *d {
		cmd := exec.Command(os.Args[0], "-close-fds", "-addr", *addr)
		serr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalln(err)
		}
		err = cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}
		s, err := ioutil.ReadAll(serr)
		s = bytes.TrimSpace(s)
		if bytes.HasPrefix(s, []byte("addr: ")) {
			fmt.Println(string(s))
			cmd.Process.Release()
		} else {
			log.Printf("unexpected response from MarGo: `%s` error: `%v`\n", s, err)
			cmd.Process.Kill()
		}
	} else {
		l, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Fprintf(os.Stderr, "addr: %s\n", l.Addr())
		if *closeFds {
			os.Stdin.Close()
			os.Stdout.Close()
			os.Stderr.Close()
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
