package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
)

func main() {

	mailserver := "mailserver:465"
	username := ""
	userpassowrd := ""
	hostname, _, _ := net.SplitHostPort(mailserver)

	from := mail.Address{Name: username, Address: username + "@" + hostname}
	to := mail.Address{Name: username, Address: username + "@" + hostname}
	subj := "This is the email subject"
	body := "This is an example body.\n With two lines."

	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         hostname,
	}

	auth := smtp.PlainAuth(
		"",
		username,
		userpassowrd,
		hostname,
	)

	smtpconnection, err := tls.Dial("tcp", mailserver, tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	smtpclient, err := smtp.NewClient(smtpconnection, hostname)
	if err != nil {
		log.Panic(err)
	}

	if err = smtpclient.Auth(auth); err != nil {
		log.Panic(err)
	}

	if err = smtpclient.Mail(from.Address); err != nil {
		log.Panic(err)
	}

	if err = smtpclient.Rcpt(to.Address); err != nil {
		log.Panic(err)
	}

	datawriter, err := smtpclient.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = datawriter.Write([]byte(message))
	if err != nil {
		log.Panic(err)
	}

	err = datawriter.Close()
	if err != nil {
		log.Panic(err)
	}

	smtpclient.Quit()
}
