package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"regexp"
	"compress/gzip"
	"bytes"
)

type HTTPRequest struct {
	Method			string
	Path			string
	Version			string
	Headers			map[string]string
	Body			string
	UserAgent		string
	ContentType		string
	ContentLength 		string
	AcceptEncoding		string
}

func main() {
		
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	
	for {
	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
		}

	setConnection(conn)
	}
}

func setConnection(conn net.Conn) {
	defer conn.Close()	

	req, err := readRequest(conn)
	if err != nil {
		fmt.Println("Error parsing: ", err)
	}

	response := makeHandler(req)
	
	conn.Write([]byte(response))
	
}	


func makeHandler(req *HTTPRequest) (string){

	var response string

	switch path := req.Path; {
	case path == "/": 
		response = "HTTP/1.1 200 OK\r\n\r\n"
	
	
	case strings.Contains(path, "/echo/"):
		message := regexp.MustCompile("echo/(\\S+)").FindStringSubmatch(path)[1]
		if strings.Contains(req.AcceptEncoding, "gzip"){
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			_, err := gz.Write([]byte(message))
			if err != nil {
				fmt.Println("Error compressing: ", err)
			}
			err = gz.Close()
			if err != nil {
				fmt.Println("Error closing: ", err)
			}
			
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-type: text/plain\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\n\r\n%s", b.Len(), b.String())
		} else {
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message)
		}

	case strings.Contains(path, "/user-agent"):
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(req.UserAgent), req.UserAgent)	

	case strings.Contains(path, "/files/") && req.Method == "GET":
		message := regexp.MustCompile("files/(\\S+)").FindStringSubmatch(path)[1]
		var file string
		file = os.Args[2] + message
		fileContent, err := os.ReadFile(file)
		if err != nil {
			response = "HTTP/1.1 404 Not Found\r\n\r\n"
			return response
		}
		
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContent), fileContent)

	case strings.Contains(path, "/files/") && req.Method == "POST":
		
		fileName := regexp.MustCompile("files/(\\S+)").FindStringSubmatch(path)[1]
		var file string
		dir := os.Args[2]
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Println("Error creating directory:", err)
			return "Error creating directory:"
		}
		file = dir + fileName

		
		fmt.Println("Before write file success")
		data := []byte(req.Body)
		
		err := os.WriteFile(file, data, 0644)
        if err != nil {
            response = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
            return response
        }
		_, err = os.Stat(file)
		if err != nil {
			fmt.Println("File not found")
		}
		fmt.Println("After write file success")

		response = "HTTP/1.1 201 Created\r\n\r\n"
	

	default:
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}
	
	return response
}

func readRequest(conn net.Conn) (*HTTPRequest, error){
	buf := make([]byte, 1024)
	var req HTTPRequest
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Errorf("Error reading conn: %s", err)
	}
	reqString := string(buf)
	lines := strings.Split(reqString, "\r\n")
	req.Method = strings.Split(lines[0], " ")[0]
	fmt.Println("Method successful")
	req.Path = strings.Split(lines[0], " ")[1]
	fmt.Println("Path successful")
	req.Version = strings.Split(lines[0], " ")[2]
	fmt.Println("Version successful")

	req.Headers= make(map[string]string)

	for _, line := range lines[1:] {
		if line == ""{
			break
		}
		parts := strings.SplitN(line,": ", 2)
		if len(parts) != 2 {
			continue
		}
	    key := parts[0]
	    value := parts[1]

	    switch key {
	    	case "User-Agent":
	        	req.UserAgent = value
	    	case "Content-Type":
	        	req.ContentType = value
	    	case "Content-Length":
	        	req.ContentLength = value
			case "Accept-Encoding":
				req.AcceptEncoding = value
	    	default:
	        req.Headers[key] = value
	    }
	}

	if req.Method == "POST" {
		req.Body = strings.Trim(lines[5], "\x00")
		fmt.Println("Body successful")
	}
	return &req, nil
}
