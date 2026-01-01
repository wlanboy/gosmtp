package main

import (
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

func main() {
	mailserver := "127.0.0.1:1025"
	username := "testuser"
	password := "testpass"

	host, port, err := net.SplitHostPort(mailserver)
	if err != nil {
		log.Fatalf("invalid mailserver address: %v", err)
	}

	from := mail.Address{Name: username, Address: username + "@" + host}
	to := mail.Address{Name: username, Address: username + "@" + host}

	subject := "This is the email subject"
	body := "This is an example body.\nWith two lines."

	msg := buildMessage(from, to, subject, body)

	fmt.Println("Sending via plain SMTP...")
	if err := sendViaPlainSMTP(host, port, username, password, from, to, msg); err != nil {
		log.Fatal("SMTP failed:", err)
	}

	fmt.Println("Email sent successfully via plain SMTP")
}

func buildMessage(from, to mail.Address, subject, body string) string {
	headers := map[string]string{
		"From":         from.String(),
		"To":           to.String(),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	sb.WriteString("\r\n" + body)

	return sb.String()
}

func sendViaPlainSMTP(host, port, username, password string, from, to mail.Address, msg string) error {
	client, err := smtp.Dial(host + ":" + port)
	if err != nil {
		return err
	}
	defer client.Close()

	// AUTH LOGIN / AUTH PLAIN
	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return err
	}

	// MAIL FROM
	if err := client.Mail(from.Address); err != nil {
		return err
	}

	// RCPT TO
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}

	// DATA
	w, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}

	return w.Close()
}
