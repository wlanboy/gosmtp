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

func imapDial(t *testing.T) (net.Conn, *bufio.Reader, *bufio.Writer) {
	t.Helper()
	conn, err := net.Dial("tcp", IMAP_ADDR)
	if err != nil {
		t.Fatalf("IMAP connect failed: %v", err)
	}
	return conn, bufio.NewReader(conn), bufio.NewWriter(conn)
}

func imapRead(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("IMAP read failed: %v", err)
	}
	return strings.TrimSpace(line)
}

func imapWrite(t *testing.T, w *bufio.Writer, s string) {
	t.Helper()
	if _, err := w.WriteString(s + "\r\n"); err != nil {
		t.Fatalf("IMAP write failed: %v", err)
	}
	w.Flush()
}

// imapLogin reads the server greeting and performs LOGIN.
func imapLogin(t *testing.T, r *bufio.Reader, w *bufio.Writer) {
	t.Helper()
	imapRead(t, r) // * OK IMAP4rev1 Service Ready
	imapWrite(t, w, "A1 LOGIN testuser testpass")
	resp := imapRead(t, r)
	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK on LOGIN, got %s", resp)
	}
}

// imapSelect sends SELECT INBOX and drains all response lines.
func imapSelect(t *testing.T, r *bufio.Reader, w *bufio.Writer, tag string) {
	t.Helper()
	imapWrite(t, w, tag+" SELECT INBOX")
	imapRead(t, r) // * N EXISTS
	imapRead(t, r) // * FLAGS (...)
	imapRead(t, r) // * OK [PERMANENTFLAGS ...]
	resp := imapRead(t, r)
	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK SELECT, got %s", resp)
	}
}

// imapReadUntil reads lines until a line starting with tag is found.
func imapReadUntil(t *testing.T, r *bufio.Reader, tag string) []string {
	t.Helper()
	var lines []string
	for {
		line := imapRead(t, r)
		lines = append(lines, line)
		if strings.HasPrefix(line, tag) {
			return lines
		}
	}
}

///////////////////////////////////////////////////////////////
// IMAP TESTS
///////////////////////////////////////////////////////////////

func TestIMAP_Login(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)
}

func TestIMAP_LoginFail(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r) // greeting
	imapWrite(t, w, "A1 LOGIN wrong wrong")
	resp := imapRead(t, r)

	if !strings.Contains(resp, "NO") {
		t.Fatalf("expected NO, got %s", resp)
	}
}

func TestIMAP_List(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)

	imapWrite(t, w, "A2 LIST \"\" \"*\"")
	listLine := imapRead(t, r)
	resp := imapRead(t, r)

	if !strings.Contains(listLine, "INBOX") {
		t.Fatalf("expected INBOX in LIST response, got %s", listLine)
	}
	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK, got %s", resp)
	}
}

func TestIMAP_Select(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)
	imapSelect(t, r, w, "A2")
}

func TestIMAP_SelectWithoutAuth(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r) // greeting
	imapWrite(t, w, "A1 SELECT INBOX")
	resp := imapRead(t, r)

	if !strings.Contains(resp, "NO") {
		t.Fatalf("expected NO without auth, got %s", resp)
	}
}

func TestIMAP_Search(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()
	smtpSendMail(t, "<a@b>", "<c@d>", "search test body")

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)
	imapSelect(t, r, w, "A2")

	imapWrite(t, w, "A3 SEARCH ALL")
	searchLine := imapRead(t, r)
	resp := imapRead(t, r)

	if !strings.Contains(searchLine, "SEARCH") {
		t.Fatalf("expected SEARCH result line, got %s", searchLine)
	}
	if !strings.Contains(searchLine, "1") {
		t.Fatalf("expected message 1 in SEARCH result, got %s", searchLine)
	}
	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK SEARCH, got %s", resp)
	}
}

func TestIMAP_Fetch(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()
	smtpSendMail(t, "<sender@example.com>", "<rcpt@example.com>", "fetch test body")

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)
	imapSelect(t, r, w, "A2")

	imapWrite(t, w, "A3 FETCH 1 (FLAGS BODY[])")
	lines := imapReadUntil(t, r, "A3")

	found := false
	for _, l := range lines {
		if strings.Contains(l, "FETCH") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected FETCH response, got: %v", lines)
	}
	if !strings.Contains(lines[len(lines)-1], "OK") {
		t.Fatalf("expected OK FETCH, got %s", lines[len(lines)-1])
	}
}

func TestIMAP_Store(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()
	smtpSendMail(t, "<a@b>", "<c@d>", "store test body")

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)
	imapSelect(t, r, w, "A2")

	imapWrite(t, w, `A3 STORE 1 +FLAGS (\Seen)`)
	fetchLine := imapRead(t, r) // * 1 FETCH (FLAGS (\Seen))
	resp := imapRead(t, r)

	if !strings.Contains(fetchLine, `\Seen`) {
		t.Fatalf("expected \\Seen flag in response, got %s", fetchLine)
	}
	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK STORE, got %s", resp)
	}

	// Verify flag persists by re-fetching.
	flags := loadFlags("1")
	found := false
	for _, f := range flags {
		if f == `\Seen` {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected \\Seen to be persisted, got %v", flags)
	}
}

func TestIMAP_UIDFetch(t *testing.T) {
	os.RemoveAll("mails")
	startMailServer()
	smtpSendMail(t, "<a@b>", "<c@d>", "uid fetch body")

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)
	imapSelect(t, r, w, "A2")

	imapWrite(t, w, "A3 UID FETCH 1 (FLAGS BODY[])")
	lines := imapReadUntil(t, r, "A3")

	if !strings.Contains(lines[len(lines)-1], "OK") {
		t.Fatalf("expected OK UID FETCH, got %s", lines[len(lines)-1])
	}
}

func TestIMAP_Idle(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)

	imapWrite(t, w, "A2 IDLE")
	resp := imapRead(t, r)
	if !strings.Contains(resp, "+ idling") {
		t.Fatalf("expected idling, got %s", resp)
	}

	imapWrite(t, w, "DONE")
	resp = imapRead(t, r)
	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK after DONE, got %s", resp)
	}
}

func TestIMAP_Logout(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r) // greeting
	imapWrite(t, w, "A1 LOGOUT")
	imapRead(t, r) // * BYE
	resp := imapRead(t, r)

	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK LOGOUT, got %s", resp)
	}
}

func TestIMAP_UnknownCommand(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapLogin(t, r, w)

	imapWrite(t, w, "A2 XYZZY")
	resp := imapRead(t, r)

	if !strings.Contains(resp, "BAD") {
		t.Fatalf("expected BAD for unknown command, got %s", resp)
	}
}
