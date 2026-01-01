package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
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

// Für Tests: einmalig Server starten
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
	select {} // block forever
}

///////////////////////////////////////////////////////////////
// SMTP SERVER
///////////////////////////////////////////////////////////////

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

func handleSMTP(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, "220 localhost ESMTP ready")

	auth := false
	var from string
	var to []string
	var dataMode bool
	var data strings.Builder

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		upper := strings.ToUpper(cmd)

		switch {
		case strings.HasPrefix(upper, "EHLO"):
			writeLine(w, "250-localhost greets you")
			writeLine(w, "250-AUTH LOGIN PLAIN")
			writeLine(w, "250 OK")

		case strings.HasPrefix(upper, "AUTH LOGIN"):
			writeLine(w, "334 VXNlcm5hbWU6") // Username:
			u64, _ := r.ReadString('\n')
			user, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(u64))

			writeLine(w, "334 UGFzc3dvcmQ6") // Password:
			p64, _ := r.ReadString('\n')
			pass, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(p64))

			if string(user) == USER && string(pass) == PASS {
				auth = true
				writeLine(w, "235 Authentication successful")
			} else {
				writeLine(w, "535 Authentication failed")
			}

		case strings.HasPrefix(upper, "AUTH PLAIN"):
			parts := strings.Split(cmd, " ")
			if len(parts) == 3 {
				decoded, _ := base64.StdEncoding.DecodeString(parts[2])
				f := strings.Split(string(decoded), "\x00")
				if len(f) == 3 && f[1] == USER && f[2] == PASS {
					auth = true
					writeLine(w, "235 Authentication successful")
				} else {
					writeLine(w, "535 Authentication failed")
				}
			} else {
				writeLine(w, "501 Syntax error")
			}

		case strings.HasPrefix(upper, "MAIL FROM:"):
			if !auth {
				writeLine(w, "530 Authentication required")
				continue
			}
			from = strings.TrimPrefix(cmd, "MAIL FROM:")
			writeLine(w, "250 OK")

		case strings.HasPrefix(upper, "RCPT TO:"):
			if !auth {
				writeLine(w, "530 Authentication required")
				continue
			}
			to = append(to, strings.TrimPrefix(cmd, "RCPT TO:"))
			writeLine(w, "250 OK")

		case upper == "DATA":
			if !auth {
				writeLine(w, "530 Authentication required")
				continue
			}
			if from == "" {
				writeLine(w, "503 Bad sequence of commands: MAIL FROM required")
				continue
			}
			if len(to) == 0 {
				writeLine(w, "503 Bad sequence of commands: RCPT TO required")
				continue
			}
			writeLine(w, "354 End data with <CR><LF>.<CR><LF>")
			dataMode = true

		case dataMode:
			if cmd == "." {
				saveMail(from, to, data.String())
				data.Reset()
				dataMode = false
				writeLine(w, "250 Message accepted")
			} else {
				data.WriteString(cmd + "\n")
			}

		case upper == "QUIT":
			writeLine(w, "221 Bye")
			return

		default:
			writeLine(w, "502 Command not implemented")
		}
	}
}

///////////////////////////////////////////////////////////////
// IMAP SERVER
///////////////////////////////////////////////////////////////

func startIMAP() {
	log.Println("IMAP listening on", IMAP_ADDR)
	ln, err := net.Listen("tcp", IMAP_ADDR)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("IMAP accept error:", err)
			continue
		}
		go handleIMAP(conn)
	}
}

func handleIMAP(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, "* OK IMAP4rev1 Service Ready")

	auth := false
	selected := ""

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}
		tag := parts[0]
		cmd := strings.ToUpper(parts[1])
		args := ""
		if len(parts) == 3 {
			args = parts[2]
		}

		switch cmd {
		case "LOGIN":
			clean := strings.ReplaceAll(args, `"`, "")
			a := strings.Fields(clean)

			if len(a) == 2 && a[0] == USER && a[1] == PASS {
				auth = true
				writeLine(w, tag+" OK LOGIN completed")
			} else {
				writeLine(w, tag+" NO LOGIN failed")
			}

		case "LIST":
			writeLine(w, `* LIST (\HasNoChildren) "/" "INBOX"`)
			writeLine(w, tag+" OK LIST completed")

		case "SELECT":
			if !auth {
				writeLine(w, tag+" NO Authenticate first")
				continue
			}
			selected = "INBOX"
			_ = selected
			count := countMails()
			writeLine(w, fmt.Sprintf("* %d EXISTS", count))
			writeLine(w, "* FLAGS (\\Seen \\Deleted \\Answered)")
			writeLine(w, "* OK [PERMANENTFLAGS (\\Seen \\Deleted \\Answered)]")
			writeLine(w, tag+" OK [READ-WRITE] SELECT completed")

		case "UID":
			if !auth {
				writeLine(w, tag+" NO Authenticate first")
				continue
			}
			handleUID(w, tag, args)

		case "FETCH":
			if !auth {
				writeLine(w, tag+" NO Authenticate first")
				continue
			}
			handleFetch(w, tag, args)

		case "STORE":
			if !auth {
				writeLine(w, tag+" NO Authenticate first")
				continue
			}
			handleStore(w, tag, args)

		case "SEARCH":
			if !auth {
				writeLine(w, tag+" NO Authenticate first")
				continue
			}
			handleSearch(w, tag)

		case "IDLE":
			if !auth {
				writeLine(w, tag+" NO Authenticate first")
				continue
			}
			writeLine(w, "+ idling")
			r.ReadString('\n')
			writeLine(w, tag+" OK IDLE terminated")

		case "LOGOUT":
			writeLine(w, "* BYE IMAP server logging out")
			writeLine(w, tag+" OK LOGOUT completed")
			return

		default:
			writeLine(w, tag+" BAD Unknown command")
		}
	}
}

///////////////////////////////////////////////////////////////
// IMAP HELPERS
///////////////////////////////////////////////////////////////

func handleUID(w *bufio.Writer, tag, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 1 {
		writeLine(w, tag+" BAD UID syntax")
		return
	}
	sub := strings.ToUpper(strings.Fields(parts[0])[0])

	switch sub {
	case "FETCH":
		if len(parts) < 2 {
			writeLine(w, tag+" BAD UID FETCH syntax")
			return
		}
		// In diesem einfachen Server: UID == Sequenznummer
		handleFetch(w, tag, parts[1])
	case "SEARCH":
		handleSearch(w, tag)
	default:
		writeLine(w, tag+" BAD UID command not supported")
	}
}

// Sequenz-Set Parser: "1", "1:*", "2:4", "1,3,5:7"
func parseSeqSet(seqSet string, max int) []int {
	var ids []int
	parts := strings.Split(seqSet, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			rangeParts := strings.Split(part, ":")
			if len(rangeParts) != 2 {
				continue
			}

			start := parseId(rangeParts[0], max)
			end := parseId(rangeParts[1], max)

			if start > end {
				start, end = end, start
			}

			for i := start; i <= end; i++ {
				if i > 0 && i <= max {
					ids = append(ids, i)
				}
			}
		} else {
			id := parseId(part, max)
			if id > 0 && id <= max {
				ids = append(ids, id)
			}
		}
	}
	return uniqueSorted(ids)
}

func parseId(s string, max int) int {
	if s == "*" {
		return max
	}
	id, _ := strconv.Atoi(s)
	return id
}

func uniqueSorted(input []int) []int {
	if len(input) == 0 {
		return input
	}
	sort.Ints(input)
	unique := make([]int, 0, len(input))
	for i, val := range input {
		if i == 0 || val != input[i-1] {
			unique = append(unique, val)
		}
	}
	return unique
}

func handleFetch(w *bufio.Writer, tag, args string) {
	fields := strings.Fields(args)
	if len(fields) < 1 {
		writeLine(w, tag+" BAD FETCH syntax")
		return
	}

	seqSet := fields[0]
	max := countMails()
	ids := parseSeqSet(seqSet, max)

	for _, id := range ids {
		msgID := strconv.Itoa(id)
		flags := loadFlags(msgID)
		bodyStr := loadMail(msgID)

		// Dynamischen Envelope generieren
		envelope := getEnvelope(bodyStr)
		body := []byte(bodyStr)

		// Antwort senden
		fmt.Fprintf(w, "* %d FETCH (UID %d FLAGS (%s) ENVELOPE %s BODY[] {%d}\r\n",
			id, id, strings.Join(flags, " "), envelope, len(body))

		w.Write(body)
		w.WriteString(")\r\n")
	}
	w.Flush()
	writeLine(w, tag+" OK FETCH completed")
}

func handleStore(w *bufio.Writer, tag, args string) {
	parts := strings.Fields(args)
	if len(parts) < 3 {
		writeLine(w, tag+" BAD STORE syntax")
		return
	}
	msgNum := parts[0]
	action := strings.ToUpper(parts[1])
	flagList := parts[2:]

	// Flags stehen in Klammern: (\Seen \Deleted)
	flagsStr := strings.Join(flagList, " ")
	flagsStr = strings.Trim(flagsStr, "()")
	newFlags := strings.Fields(flagsStr)

	current := loadFlags(msgNum)

	// Helper: Flag in Slice?
	has := func(list []string, f string) bool {
		for _, x := range list {
			if x == f {
				return true
			}
		}
		return false
	}

	switch {
	case action == "FLAGS":
		// Replace
		current = newFlags

	case action == "+FLAGS" || action == "+FLAGS.SILENT":
		for _, f := range newFlags {
			if !has(current, f) {
				current = append(current, f)
			}
		}

	case action == "-FLAGS" || action == "-FLAGS.SILENT":
		var updated []string
		for _, f := range current {
			if !has(newFlags, f) {
				updated = append(updated, f)
			}
		}
		current = updated

	default:
		writeLine(w, tag+" BAD STORE action")
		return
	}

	saveFlags(msgNum, current)

	silent := strings.Contains(action, ".SILENT")
	if !silent {
		writeLine(w, fmt.Sprintf("* %s FETCH (FLAGS (%s))", msgNum, strings.Join(current, " ")))
	}

	writeLine(w, tag+" OK STORE completed")
}

func handleSearch(w *bufio.Writer, tag string) {
	count := countMails()
	var ids []string
	for i := 1; i <= count; i++ {
		ids = append(ids, fmt.Sprintf("%d", i))
	}
	writeLine(w, "* SEARCH "+strings.Join(ids, " "))
	writeLine(w, tag+" OK SEARCH completed")
}

///////////////////////////////////////////////////////////////
// STORAGE
///////////////////////////////////////////////////////////////

func saveMail(from string, to []string, raw string) {
	id := nextID()
	emlPath := filepath.Join("mails", fmt.Sprintf("%d.eml", id))
	flagsPath := filepath.Join("mails", fmt.Sprintf("%d.flags", id))

	content := raw
	// Prüfen, ob bereits Header vorhanden sind.
	// Wenn "Subject:" fehlt, bauen wir einen Header-Block.
	if !strings.Contains(strings.ToUpper(raw), "SUBJECT:") {
		header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Mock Mail %d\r\nDate: %s\r\n\r\n",
			from, strings.Join(to, ", "), id, time.Now().Format(time.RFC1123Z))
		content = header + raw
	} else {
		// Wenn Header da sind, aber die Leerzeile zum Body fehlt:
		// Wir suchen nach der ersten Zeile, die kein Header-Format (Key: Value) hat.
		lines := strings.Split(raw, "\n")
		headerEnded := false
		var fixedContent strings.Builder
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !headerEnded && trimmed != "" && !strings.Contains(line, ":") {
				// Hier beginnt der Body, aber es gab keine Leerzeile!
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

	// Vereinheitliche auf CRLF (IMAP Standard)
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
		// Minimaler Fallback-Envelope, damit der Client nicht crasht
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
		subject := msg.Header.Get("Subject")
		if subject != "" {
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
		base := filepath.Base(f) // "12.eml"
		idStr := strings.TrimSuffix(base, ".eml")
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
