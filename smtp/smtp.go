package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/corpix/smtpd"
	uuid "github.com/satori/go.uuid"
)

type smtpServer struct{}

func main() {
	var err error
	var socket string = "127.0.0.1:1025"
	fmt.Println("SMTP server: " + socket)

	c, err := net.Listen("tcp", socket)
	if err != nil {
		panic(err)
	}

	for {
		err = smtpd.Serve(c, &smtpServer{})
		if err != nil {
			fmt.Println("Socket Error: ", err)
		}
	}
}

/*
* swaks --server 127.0.0.1:1025 -f testfrom@wlanboy.com -t testto@wlanboy.com
 */
func (s *smtpServer) ServeSMTP(c net.Conn, e *smtpd.Envelope) {

	id, _ := uuid.NewV4()
	msg, err := e.Message()
	if err != nil {
		fmt.Println("Message Error: ", err)
	}

	body, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		fmt.Println("Body Error: ", err)
	}

	from := e.From
	to := e.To

	writeToFile(id.String(), from, to, body)
}

func writeToFile(id string, from string, to []string, body []byte) {
	fo, err := os.Create(id + ".txt")
	if err != nil {
		fmt.Println("File Error: ", err)
	}

	fmt.Println("From: ", from)
	fo.WriteString(from + "\n")

	for _, line := range to {
		fmt.Println("To: ", line)
		fo.WriteString(line + "\n")
	}

	fmt.Println("Body: ")
	fmt.Print(string(body))
	fo.WriteString(string(body))

	defer func() {
		if err := fo.Close(); err != nil {
			fmt.Println("File Close Error: ", err)
		}
	}()
}
