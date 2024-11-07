package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	BUFF_SIZE = 2048
	CRLF      = "\r\n"
)

var (
	ErrNoRequest     = errors.New("No http request provided")
	ErrInvalidMethod = errors.New("Invalid http method")

	Methods = map[string]struct{}{
		"GET":    struct{}{},
		"POST":   struct{}{},
		"PUT":    struct{}{},
		"DELETE": struct{}{},
	}
)

var _ = net.Listen
var _ = os.Exit

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
		}
		fmt.Println("Conn established")
		go HandleConn(conn)
	}
}

func HandleConn(c net.Conn) error {
	defer c.Close()

	buffer := make([]byte, BUFF_SIZE)
	nread, err := c.Read(buffer)
	if err != nil {
		return err
	}
	lines := strings.Split(string(buffer), CRLF)
	if nread == 0 || len(lines) < 2 {
		return ErrNoRequest
	}

	lines[0] = strings.TrimSpace(lines[0])

	request := strings.Fields(lines[0])
	// fmt.Println(request[0])
	// fmt.Println(request[1])
	// fmt.Println(request[2])

	if _, ok := Methods[request[0]]; !ok {
		return ErrInvalidMethod
	}

	if request[1] == "/" {
		fmt.Fprintf(c, "HTTP/1.1 200 OK\r\n\r\n")
	} else {
		fmt.Fprintf(c, "HTTP/1.1 404 Not Found\r\n\r\n")
	}

	return nil
}
