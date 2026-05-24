package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	SMTP_ADDR = "127.0.0.1:1025"
	IMAP_ADDR = "127.0.0.1:1143"

	USER = "testuser"
	PASS = "testpass"
)

var serverStarted = false

func startMailServer() {
	if err := os.MkdirAll("mails", 0755); err != nil {
		log.Fatal(err)
	}

	if serverStarted {
		return
	}
	serverStarted = true

	go startSMTP()
	go startIMAP()

	time.Sleep(300 * time.Millisecond)
}

func main() {
	startMailServer()
	select {}
}

func startSMTP() {
	log.Println("SMTP listening on", SMTP_ADDR)
	ln, err := net.Listen("tcp", SMTP_ADDR)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("SMTP accept error:", err)
			continue
		}
		go handleSMTP(conn)
	}
}

///////////////////////////////////////////////////////////////
// STORAGE
///////////////////////////////////////////////////////////////

func saveMail(from string, to []string, raw string) {
	id := nextID()
	emlPath := filepath.Join("mails", fmt.Sprintf("%d.eml", id))
	flagsPath := filepath.Join("mails", fmt.Sprintf("%d.flags", id))

	content := raw
	if !strings.Contains(strings.ToUpper(raw), "SUBJECT:") {
		header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Mock Mail %d\r\nDate: %s\r\n\r\n",
			from, strings.Join(to, ", "), id, time.Now().Format(time.RFC1123Z))
		content = header + raw
	} else {
		lines := strings.Split(raw, "\n")
		headerEnded := false
		var fixedContent strings.Builder
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !headerEnded && trimmed != "" && !strings.Contains(line, ":") {
				fixedContent.WriteString("\r\n")
				headerEnded = true
			}
			if trimmed == "" {
				headerEnded = true
			}
			fixedContent.WriteString(line + "\n")
		}
		content = fixedContent.String()
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\n", "\r\n")

	os.WriteFile(emlPath, []byte(content), 0644)
	os.WriteFile(flagsPath, []byte(""), 0644)
	log.Println("Saved mail:", emlPath)
}

func getEnvelope(raw string) string {
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		log.Printf("Envelope Parse Error: %v", err)
		return `(NIL "Format Error: Missing Blank Line" NIL NIL NIL NIL NIL NIL NIL NIL)`
	}

	formatAddr := func(headerName string) string {
		addrList, _ := msg.Header.AddressList(headerName)
		if len(addrList) == 0 {
			return "NIL"
		}
		var parts []string
		for _, a := range addrList {
			name := "NIL"
			if a.Name != "" {
				name = fmt.Sprintf(`"%s"`, a.Name)
			}
			atIdx := strings.Index(a.Address, "@")
			mailbox, host := "user", "unknown"
			if atIdx != -1 {
				mailbox = a.Address[:atIdx]
				host = a.Address[atIdx+1:]
			}
			parts = append(parts, fmt.Sprintf(`(%s NIL "%s" "%s")`, name, mailbox, host))
		}
		return "(" + strings.Join(parts, " ") + ")"
	}

	date := msg.Header.Get("Date")
	if date == "" {
		date = time.Now().Format(time.RFC1123Z)
	}
	subject := msg.Header.Get("Subject")
	if subject == "" {
		subject = "No Subject"
	}

	return fmt.Sprintf(`("%s" "%s" %s %s %s %s NIL NIL NIL NIL)`,
		date, subject, formatAddr("From"), formatAddr("From"),
		formatAddr("Reply-To"), formatAddr("To"))
}

func loadFlags(id string) []string {
	f := filepath.Join("mails", id+".flags")
	raw, err := os.ReadFile(f)
	if err != nil {
		return []string{}
	}
	return strings.Fields(string(raw))
}

func saveFlags(id string, flags []string) {
	f := filepath.Join("mails", id+".flags")
	if err := os.WriteFile(f, []byte(strings.Join(flags, " ")), 0644); err != nil {
		log.Println("write flags error:", err)
	}
}

func loadMail(id string) string {
	f := filepath.Join("mails", id+".eml")
	raw, err := os.ReadFile(f)
	if err != nil {
		return ""
	}

	if msg, err := mail.ReadMessage(strings.NewReader(string(raw))); err == nil {
		if subject := msg.Header.Get("Subject"); subject != "" {
			log.Println("Loaded mail", id, "Subject:", subject)
		}
	}

	return string(raw)
}

func countMails() int {
	files, _ := filepath.Glob("mails/*.eml")
	return len(files)
}

func nextID() int {
	files, _ := filepath.Glob("mails/*.eml")
	max := 0
	for _, f := range files {
		idStr := strings.TrimSuffix(filepath.Base(f), ".eml")
		id, _ := strconv.Atoi(idStr)
		if id > max {
			max = id
		}
	}
	return max + 1
}

///////////////////////////////////////////////////////////////
// UTIL
///////////////////////////////////////////////////////////////

func writeLine(w *bufio.Writer, msg string) {
	w.WriteString(msg + "\r\n")
	w.Flush()
}
