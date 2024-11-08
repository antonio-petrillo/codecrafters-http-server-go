package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	BUFF_SIZE = 4096
)

var (
	InvalidData          = errors.New("Invalid data")
	InvalidRequest       = errors.New("Invalid request")
	InvalidMethod        = errors.New("Invalid method")
	InvalidURL           = errors.New("Invalid URL")
	InvalidHeader        = errors.New("Invalid header")
	InvalidContentLength = errors.New("Invalid content length")

	Methods = map[string]struct{}{
		"GET":    struct{}{},
		"POST":   struct{}{},
		"PUT":    struct{}{},
		"DELETE": struct{}{},
	}
)

type Request struct {
	Version       string
	Method        string
	Url           *url.URL
	Headers       map[string]string
	ContentLength int
	Body          []byte
}

func ParseRequest(raw []byte) (*Request, error) {
	lines := strings.Split(string(raw), "\r\n")
	if len(lines) == 0 {
		return nil, InvalidData
	}

	req := strings.Fields(lines[0])
	if len(req) != 3 {
		return nil, InvalidRequest
	}

	if _, ok := Methods[req[0]]; !ok {
		return nil, InvalidMethod
	}

	url, err := url.Parse(req[1])
	if err != nil {
		return nil, InvalidURL
	}
	lines = lines[1:]
	content := []byte{}
	length := 0
	headers := make(map[string]string)
	for i, line := range lines {
		if len(line) == 0 {
			continue // skip \r\n\r\n
		}
		if i == len(lines)-1 {
			content = []byte(line)
		} else {
			headerKey, headerValue, ok := strings.Cut(line, ": ")
			if ok {
				headers[headerKey] = headerValue
				if headerKey == "Content-Length" {
					length, err = strconv.Atoi(headerValue)
					if err != nil {
						return nil, InvalidContentLength
					}
				}
			} else { // invalid header
				log.Println("Header", line)
				return nil, InvalidHeader
			}
		}
	}

	return &Request{
		Version:       req[2],
		Method:        req[0],
		Url:           url,
		Headers:       headers,
		ContentLength: length,
		Body:          content,
	}, nil
}

func SendResponse(w io.Writer, status int, headers map[string]string, contentLength int, body []byte) {
	wb := bufio.NewWriter(w)
	msg := getStatusMessage(status)
	fmt.Fprintf(wb, "HTTP/1.1 %d %s\r\n", status, msg) // HTTP/1.1 200 OK\r\n
	for k, v := range headers {
		fmt.Fprintf(wb, "%s: %s\r\n", k, v) // i.e. Content-Type: text/plain\r\n
	}
	if contentLength > 0 {
		fmt.Fprintf(wb, "Content-Length: %d\r\n\r\n", contentLength)
	} else {
		fmt.Fprintf(wb, "\r\n")
	}
	wb.Write(body[0:contentLength])
	wb.Flush()
}

// Just a few
func getStatusMessage(status int) string {
	switch status {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	default:
		return "Unkwown"
	}
}

func HandleConn(c net.Conn) {
	log.Println("New connection handling")
	defer c.Close()

	buffer := make([]byte, BUFF_SIZE)
	nread, err := c.Read(buffer)
	if err != nil || nread == 0 {
		return
	}

	req, err := ParseRequest(buffer)
	if err != nil {
		log.Println(err.Error())
		SendResponse(c, 400, make(map[string]string), 0, []byte{})
		return
	}

	path := req.Url.Path
	log.Printf("Requested %q %s", req.Method, path)

	if path == "/" {
		SendResponse(c, 200, make(map[string]string), 0, []byte{})
	} else if strings.HasPrefix(path, "/echo/") {
		echo, _ := strings.CutPrefix(path, "/echo/")
		binEcho := []byte(echo)
		headers := map[string]string{
			"Content-Type": "text/plain",
		}
		SendResponse(c, 200, headers, len(binEcho), binEcho)
	} else if strings.HasPrefix(path, "/user-agent") {
		headers := map[string]string{
			"Content-Type": "text/plain",
		}
		binUa := []byte{}
		if ua, ok := req.Headers["User-Agent"]; ok {
			binUa = []byte(ua)
		}
		SendResponse(c, 200, headers, len(binUa), binUa)
	} else if strings.HasPrefix(path, "/files/") {
		filename, _ := strings.CutPrefix(path, "/files/")
		if len(filename) == 0 {
			SendResponse(c, 404, make(map[string]string), 0, []byte{})
		}
		switch req.Method {
		case "GET":
			SendResponseFileGet(c, filename)
		case "POST":
			SendResponseFilePost(c, filename, req)
		default:
			SendResponse(c, 405, make(map[string]string), 0, []byte{})
		}
	} else {
		SendResponse(c, 404, map[string]string{}, 0, []byte{})
	}
}

func SendResponseFilePost(c net.Conn, filename string, req *Request) {
	filepath := fmt.Sprintf("%s/%s", staticDir, filename)
	file, err := os.OpenFile(filepath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		SendResponse(c, 500, map[string]string{}, 0, []byte{})
		return
	}
	file.Write(req.Body[0:req.ContentLength])
	file.Close()
	SendResponse(c, 201, map[string]string{}, 0, []byte{})
}

func SendResponseFileGet(c net.Conn, filename string) {
	filepath := fmt.Sprintf("%s/%s", staticDir, filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		SendResponse(c, 404, map[string]string{}, 0, []byte{})
		return
	}
	headers := map[string]string{
		"Content-Type": "application/octet-stream",
	}
	SendResponse(c, 200, headers, len(content), content)
}
