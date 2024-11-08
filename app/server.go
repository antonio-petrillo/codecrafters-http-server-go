package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	BUFF_SIZE = 2048
	CRLF      = "\r\n"
)

var (
	ErrNoRequest     = errors.New("No http request provided")
	ErrInvalidMethod = errors.New("Invalid http method")

	staticDir = "./"

	Methods = map[string]struct{}{
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

// TODO: implements handlers as a Trie

func HandleConn(c net.Conn) {
	defer c.Close()
	log.Println("New Connection handling")

	buffer := make([]byte, BUFF_SIZE)
	nread, err := c.Read(buffer)
	if err != nil {
		return
	}
	lines := strings.Split(string(buffer), CRLF)
	if nread == 0 || len(lines) < 2 {
		return
	}

	lines[0] = strings.TrimSpace(lines[0])

	request := strings.Fields(lines[0])

	if _, ok := Methods[request[0]]; !ok {
		return
	}

	if request[1] == "/" {
		log.Println("Requested root \"/\"")
		fmt.Fprintf(c, "HTTP/1.1 200 OK\r\n\r\n")
	} else if strings.HasPrefix(request[1], "/echo/") {
		echo, _ := strings.CutPrefix(request[1], "/echo/")
		log.Printf("Requested echo /echo/%s", echo)

		w := bufio.NewWriter(c)
		w.WriteString("HTTP/1.1 200 OK\r\n")
		w.WriteString("Content-Type: text/plain\r\n")
		w.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(echo)))
		w.WriteString(fmt.Sprintf("\r\n%s", echo))
		w.Flush()

	} else if request[1] == "/user-agent" {
		log.Printf("Requested /user-agent")
		w := bufio.NewWriter(c)
		w.WriteString("HTTP/1.1 200 OK\r\n")
		w.WriteString("Content-Type: text/plain\r\n")
		userAgent := ""
		for _, line := range lines[1:] {
			log.Println(line)
			if strings.HasPrefix(line, "User-Agent:") {
				userAgent, _ = strings.CutPrefix(line, "User-Agent:")
				userAgent = strings.TrimSpace(userAgent)
				break
			}
		}
		if userAgent != "" {
			w.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(userAgent)))
			w.WriteString(fmt.Sprintf("\r\n%s", userAgent))
		}
		w.Flush()

	} else if strings.HasPrefix(request[1], "/files/") {
		filename, _ := strings.CutPrefix(request[1], "/files/")
		if len(filename) == 0 {
			fmt.Fprintf(c, "HTTP/1.1 404 Not Found\r\n\r\n")
		}
		if request[0] == "GET" {
			log.Printf("Requested GET /files/")
			handleGetFiles(c, filename)
		} else if request[0] == "POST" {
			log.Printf("Requested POST /files/")
			content := []byte{}
			size := 0
			grab_next := false
			for _, line := range lines[1:] {
				if strings.HasPrefix(line, "Content-Length:") {
					size_str, _ := strings.CutPrefix(line, "Content-Length:")
					size_str = strings.TrimSpace(size_str)

					size_, err := strconv.Atoi(size_str)
					if err != nil {
						fmt.Fprintf(c, "HTTP/1.1 400 Bad Request\r\n\r\n")
						return
					}
					size = size_
				}
				if grab_next {
					content = []byte(line)
					break
				}
				if len(line) == 0 {
					grab_next = true
				}
			}
			handlePostFiles(c, filename, content, size)
		}
	} else {
		log.Println("Requested not found")
		fmt.Fprintf(c, "HTTP/1.1 404 Not Found\r\n\r\n")
	}

}

func handlePostFiles(c net.Conn, filename string, content []byte, size int) {
	filepath := fmt.Sprintf("%s/%s", staticDir, filename)
	file, err := os.OpenFile(filepath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(c, "HTTP/1.1 500 Internal Server Error\r\n\r\n")
	}
	log.Printf("Content:%s\n", string(content[0:size]))
	file.Write(content[0:size])
	file.Close()
	fmt.Fprintf(c, "HTTP/1.1 201 Created\r\n\r\n")
}

func handleGetFiles(c net.Conn, filename string) {
	filepath := fmt.Sprintf("%s/%s", staticDir, filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Println("File not found")
		fmt.Fprintf(c, "HTTP/1.1 404 Not Found\r\n\r\n")
	}
	log.Println("File found")

	w := bufio.NewWriter(c)
	w.WriteString("HTTP/1.1 200 OK\r\n")
	w.WriteString("Content-Type: application/octet-stream\r\n")
	w.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content)))
	w.Write(content)
	w.Flush()
}
