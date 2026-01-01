package main

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

func imapDial(t *testing.T) (net.Conn, *bufio.Reader, *bufio.Writer) {
	conn, err := net.Dial("tcp", IMAP_ADDR)
	if err != nil {
		t.Fatalf("IMAP connect failed: %v", err)
	}
	return conn, bufio.NewReader(conn), bufio.NewWriter(conn)
}

func imapRead(t *testing.T, r *bufio.Reader) string {
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("IMAP read failed: %v", err)
	}
	return strings.TrimSpace(line)
}

func imapWrite(t *testing.T, w *bufio.Writer, s string) {
	_, err := w.WriteString(s + "\r\n")
	if err != nil {
		t.Fatalf("IMAP write failed: %v", err)
	}
	w.Flush()
}

///////////////////////////////////////////////////////////////
// IMAP TESTS
///////////////////////////////////////////////////////////////

func TestIMAP_Login(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r)

	imapWrite(t, w, "A1 LOGIN testuser testpass")
	resp := imapRead(t, r)

	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK, got %s", resp)
	}
}

func TestIMAP_LoginFail(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r)

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

	imapRead(t, r)

	imapWrite(t, w, "A1 LOGIN testuser testpass")
	imapRead(t, r)

	imapWrite(t, w, "A2 LIST \"\" \"*\"")
	imapRead(t, r)
	resp := imapRead(t, r)

	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK, got %s", resp)
	}
}

func TestIMAP_Select(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r)

	imapWrite(t, w, "A1 LOGIN testuser testpass")
	imapRead(t, r)

	imapWrite(t, w, "A2 SELECT INBOX")
	imapRead(t, r)
	imapRead(t, r)
	resp := imapRead(t, r)

	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK, got %s", resp)
	}
}

func TestIMAP_Idle(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r)

	imapWrite(t, w, "A1 LOGIN testuser testpass")
	imapRead(t, r)

	imapWrite(t, w, "A2 IDLE")
	resp := imapRead(t, r)

	if !strings.Contains(resp, "+ idling") {
		t.Fatalf("expected idling, got %s", resp)
	}

	imapWrite(t, w, "DONE")
	resp = imapRead(t, r)

	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK, got %s", resp)
	}
}

func TestIMAP_Logout(t *testing.T) {
	startMailServer()

	conn, r, w := imapDial(t)
	defer conn.Close()

	imapRead(t, r)

	imapWrite(t, w, "A1 LOGOUT")
	imapRead(t, r)
	resp := imapRead(t, r)

	if !strings.Contains(resp, "OK") {
		t.Fatalf("expected OK, got %s", resp)
	}
}
