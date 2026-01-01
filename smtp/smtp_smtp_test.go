package main

import (
	"bufio"
	"net"
	"os"
	"strings"
	"testing"
)

func smtpDial(t *testing.T) (net.Conn, *bufio.Reader, *bufio.Writer) {
	conn, err := net.Dial("tcp", SMTP_ADDR)
	if err != nil {
		t.Fatalf("SMTP connect failed: %v", err)
	}
	return conn, bufio.NewReader(conn), bufio.NewWriter(conn)
}

func smtpRead(t *testing.T, r *bufio.Reader) string {
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("SMTP read failed: %v", err)
	}
	return strings.TrimSpace(line)
}

func smtpWrite(t *testing.T, w *bufio.Writer, s string) {
	_, err := w.WriteString(s + "\r\n")
	if err != nil {
		t.Fatalf("SMTP write failed: %v", err)
	}
	w.Flush()
}

///////////////////////////////////////////////////////////////
// SMTP TESTS
///////////////////////////////////////////////////////////////

func TestSMTP_HappyPath(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner

	smtpWrite(t, w, "EHLO localhost")
	smtpRead(t, r)
	smtpRead(t, r)
	smtpRead(t, r)

	smtpWrite(t, w, "AUTH LOGIN")
	smtpRead(t, r)
	smtpWrite(t, w, "dGVzdHVzZXI=") // testuser
	smtpRead(t, r)
	smtpWrite(t, w, "dGVzdHBhc3M=") // testpass
	smtpRead(t, r)

	smtpWrite(t, w, "MAIL FROM:<a@b>")
	smtpRead(t, r)

	smtpWrite(t, w, "RCPT TO:<c@d>")
	smtpRead(t, r)

	smtpWrite(t, w, "DATA")
	smtpRead(t, r)

	smtpWrite(t, w, "Hello World")
	smtpWrite(t, w, ".")
	smtpRead(t, r)

	if countMails() != 1 {
		t.Fatalf("expected 1 mail, got %d", countMails())
	}
}

func TestSMTP_AuthFail(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r)

	smtpWrite(t, w, "AUTH LOGIN")
	smtpRead(t, r)
	smtpWrite(t, w, "d3Jvbmc=") // wrong
	smtpRead(t, r)
	smtpWrite(t, w, "d3Jvbmc=") // wrong
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "535") {
		t.Fatalf("expected 535 auth fail, got %s", resp)
	}
}

func TestSMTP_NoAuthMailFrom(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r)

	smtpWrite(t, w, "MAIL FROM:<a@b>")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "530") {
		t.Fatalf("expected 530, got %s", resp)
	}
}

func TestSMTP_NoAuthRcptTo(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r)

	smtpWrite(t, w, "RCPT TO:<a@b>")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "530") {
		t.Fatalf("expected 530, got %s", resp)
	}
}

func TestSMTP_DataWithoutMailFrom(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r)

	// AUTH first
	smtpWrite(t, w, "AUTH LOGIN")
	smtpRead(t, r)
	smtpWrite(t, w, "dGVzdHVzZXI=")
	smtpRead(t, r)
	smtpWrite(t, w, "dGVzdHBhc3M=")
	smtpRead(t, r)

	// DATA without MAIL FROM
	smtpWrite(t, w, "DATA")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "503") {
		t.Fatalf("expected 503, got %s", resp)
	}
}
