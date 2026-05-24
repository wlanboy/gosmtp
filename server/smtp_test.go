package main

import (
	"bufio"
	"net"
	"os"
	"strings"
	"testing"
)

///////////////////////////////////////////////////////////////
// HELPERS
///////////////////////////////////////////////////////////////

func smtpDial(t *testing.T) (net.Conn, *bufio.Reader, *bufio.Writer) {
	t.Helper()
	conn, err := net.Dial("tcp", SMTP_ADDR)
	if err != nil {
		t.Fatalf("SMTP connect failed: %v", err)
	}
	return conn, bufio.NewReader(conn), bufio.NewWriter(conn)
}

func smtpRead(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("SMTP read failed: %v", err)
	}
	return strings.TrimSpace(line)
}

func smtpWrite(t *testing.T, w *bufio.Writer, s string) {
	t.Helper()
	if _, err := w.WriteString(s + "\r\n"); err != nil {
		t.Fatalf("SMTP write failed: %v", err)
	}
	w.Flush()
}

// smtpLogin performs banner read, EHLO and AUTH LOGIN.
func smtpLogin(t *testing.T, r *bufio.Reader, w *bufio.Writer) {
	t.Helper()
	smtpRead(t, r) // 220 banner
	smtpWrite(t, w, "EHLO localhost")
	smtpRead(t, r) // 250-localhost greets you
	smtpRead(t, r) // 250-AUTH LOGIN PLAIN
	smtpRead(t, r) // 250 OK
	smtpWrite(t, w, "AUTH LOGIN")
	smtpRead(t, r)                    // 334 Username:
	smtpWrite(t, w, "dGVzdHVzZXI=") // testuser
	smtpRead(t, r)                    // 334 Password:
	smtpWrite(t, w, "dGVzdHBhc3M=") // testpass
	resp := smtpRead(t, r)            // 235
	if !strings.Contains(resp, "235") {
		t.Fatalf("expected 235 auth success, got %s", resp)
	}
}

// smtpSendMail sends a single mail and closes the connection.
func smtpSendMail(t *testing.T, from, to, body string) {
	t.Helper()
	conn, r, w := smtpDial(t)
	defer conn.Close()
	smtpLogin(t, r, w)
	smtpWrite(t, w, "MAIL FROM:"+from)
	smtpRead(t, r) // 250 OK
	smtpWrite(t, w, "RCPT TO:"+to)
	smtpRead(t, r) // 250 OK
	smtpWrite(t, w, "DATA")
	smtpRead(t, r) // 354
	smtpWrite(t, w, body)
	smtpWrite(t, w, ".")
	resp := smtpRead(t, r) // 250 Message accepted
	if !strings.Contains(resp, "250") {
		t.Fatalf("expected 250 after DATA, got %s", resp)
	}
}

///////////////////////////////////////////////////////////////
// SMTP TESTS
///////////////////////////////////////////////////////////////

func TestSMTP_HappyPath(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()

	smtpSendMail(t, "<sender@example.com>", "<rcpt@example.com>", "Hello World")

	if countMails() != 1 {
		t.Fatalf("expected 1 mail, got %d", countMails())
	}
}

func TestSMTP_AuthFail(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
	smtpWrite(t, w, "AUTH LOGIN")
	smtpRead(t, r)                   // 334 Username:
	smtpWrite(t, w, "d3Jvbmc=")    // wrong
	smtpRead(t, r)                   // 334 Password:
	smtpWrite(t, w, "d3Jvbmc=")    // wrong
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "535") {
		t.Fatalf("expected 535 auth fail, got %s", resp)
	}
}

func TestSMTP_AuthPlain(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
	// AUTH PLAIN with inline credentials: \0testuser\0testpass → base64
	smtpWrite(t, w, "AUTH PLAIN AHRlc3R1c2VyAHRlc3RwYXNz")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "235") {
		t.Fatalf("expected 235 auth success, got %s", resp)
	}
}

func TestSMTP_HELO(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
	smtpWrite(t, w, "HELO localhost")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "250") {
		t.Fatalf("expected 250, got %s", resp)
	}
}

func TestSMTP_NOOP(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
	smtpWrite(t, w, "NOOP")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "250") {
		t.Fatalf("expected 250, got %s", resp)
	}
}

func TestSMTP_RSET(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpLogin(t, r, w)

	smtpWrite(t, w, "MAIL FROM:<a@b>")
	smtpRead(t, r) // 250 OK

	smtpWrite(t, w, "RSET")
	resp := smtpRead(t, r)
	if !strings.Contains(resp, "250") {
		t.Fatalf("expected 250 after RSET, got %s", resp)
	}

	// After RSET, DATA must fail because MAIL FROM was cleared.
	smtpWrite(t, w, "DATA")
	resp = smtpRead(t, r)
	if !strings.Contains(resp, "503") {
		t.Fatalf("expected 503 after RSET+DATA, got %s", resp)
	}
}

func TestSMTP_Quit(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
	smtpWrite(t, w, "QUIT")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "221") {
		t.Fatalf("expected 221, got %s", resp)
	}
}

func TestSMTP_UnknownCommand(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
	smtpWrite(t, w, "XYZZY")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "502") {
		t.Fatalf("expected 502, got %s", resp)
	}
}

func TestSMTP_NoAuthMailFrom(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpRead(t, r) // banner
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

	smtpRead(t, r) // banner
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

	smtpLogin(t, r, w)

	smtpWrite(t, w, "DATA")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "503") {
		t.Fatalf("expected 503, got %s", resp)
	}
}

func TestSMTP_DataWithoutRcptTo(t *testing.T) {
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpLogin(t, r, w)

	smtpWrite(t, w, "MAIL FROM:<a@b>")
	smtpRead(t, r) // 250 OK

	smtpWrite(t, w, "DATA")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "503") {
		t.Fatalf("expected 503, got %s", resp)
	}
}

func TestSMTP_MultipleRecipients(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpLogin(t, r, w)

	smtpWrite(t, w, "MAIL FROM:<sender@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "RCPT TO:<alice@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "RCPT TO:<bob@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "RCPT TO:<carol@example.com>")
	smtpRead(t, r)

	smtpWrite(t, w, "DATA")
	smtpRead(t, r)
	smtpWrite(t, w, "Hello three recipients")
	smtpWrite(t, w, ".")
	resp := smtpRead(t, r)

	if !strings.Contains(resp, "250") {
		t.Fatalf("expected 250, got %s", resp)
	}
	if countMails() != 1 {
		t.Fatalf("expected 1 mail, got %d", countMails())
	}
}

func TestSMTP_DotStuffing(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpLogin(t, r, w)

	smtpWrite(t, w, "MAIL FROM:<a@b>")
	smtpRead(t, r)
	smtpWrite(t, w, "RCPT TO:<c@d>")
	smtpRead(t, r)
	smtpWrite(t, w, "DATA")
	smtpRead(t, r)

	// RFC 5321: leading dot is escaped with an extra dot by the client.
	smtpWrite(t, w, "..dotline")
	smtpWrite(t, w, "normal line")
	smtpWrite(t, w, ".")
	smtpRead(t, r) // 250 Message accepted

	content := loadMail("1")
	if !strings.Contains(content, ".dotline") {
		t.Fatalf("expected unstuffed '.dotline' in stored mail, got:\n%s", content)
	}
	if strings.Contains(content, "..dotline") {
		t.Fatalf("double dot was not unstuffed in stored mail")
	}
}

func TestSMTP_TwoMailsInOneSession(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()

	conn, r, w := smtpDial(t)
	defer conn.Close()

	smtpLogin(t, r, w)

	// First mail
	smtpWrite(t, w, "MAIL FROM:<first@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "RCPT TO:<rcpt1@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "DATA")
	smtpRead(t, r)
	smtpWrite(t, w, "First mail body")
	smtpWrite(t, w, ".")
	smtpRead(t, r)

	// Second mail in the same authenticated session
	smtpWrite(t, w, "MAIL FROM:<second@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "RCPT TO:<rcpt2@example.com>")
	smtpRead(t, r)
	smtpWrite(t, w, "DATA")
	smtpRead(t, r)
	smtpWrite(t, w, "Second mail body")
	smtpWrite(t, w, ".")
	smtpRead(t, r)

	if countMails() != 2 {
		t.Fatalf("expected 2 mails, got %d", countMails())
	}
}
