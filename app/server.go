package main

import (
	"errors"
	"log"
	"net"
	"os"
	"strings"
)

const (
	BUFF_SIZEIa = 2048
	CRLF        = "\r\n"
)

var (
	ErrNoRequest     = errors.New("No http request provided")
	ErrInvalidMethod = errors.New("Invalid http method")

	staticDir = "./"

	methods = map[string]struct{}{
		"GET":    struct{}{},
		"POST":   struct{}{},
		"PUT":    struct{}{},
		"DELETE": struct{}{},
	}
)

func main() {
	for i, arg := range os.Args {
		if arg == "--directory" {
			if i == len(os.Args)-1 {
				log.Println("Missing directory param")
				os.Exit(1)
			}
			staticDir = strings.TrimRight(os.Args[i+1], "/")
			break
		}
	}

	log.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting connection: ", err.Error())
		}
		go HandleConn(conn)
	}
}
