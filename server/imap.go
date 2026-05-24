package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
)

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
		handleFetch(w, tag, parts[1])
	case "SEARCH":
		handleSearch(w, tag)
	default:
		writeLine(w, tag+" BAD UID command not supported")
	}
}

// parseSeqSet parses IMAP sequence sets: "1", "1:*", "2:4", "1,3,5:7"
func parseSeqSet(seqSet string, max int) []int {
	var ids []int
	for _, part := range strings.Split(seqSet, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			rangeParts := strings.Split(part, ":")
			if len(rangeParts) != 2 {
				continue
			}
			start := parseSeqID(rangeParts[0], max)
			end := parseSeqID(rangeParts[1], max)
			if start > end {
				start, end = end, start
			}
			for i := start; i <= end; i++ {
				if i > 0 && i <= max {
					ids = append(ids, i)
				}
			}
		} else {
			id := parseSeqID(part, max)
			if id > 0 && id <= max {
				ids = append(ids, id)
			}
		}
	}
	return uniqueSorted(ids)
}

func parseSeqID(s string, max int) int {
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

	max := countMails()
	ids := parseSeqSet(fields[0], max)

	for _, id := range ids {
		msgID := strconv.Itoa(id)
		flags := loadFlags(msgID)
		bodyStr := loadMail(msgID)
		envelope := getEnvelope(bodyStr)
		body := []byte(bodyStr)

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
	flagsStr := strings.Trim(strings.Join(parts[2:], " "), "()")
	newFlags := strings.Fields(flagsStr)

	current := loadFlags(msgNum)

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

	if !strings.Contains(action, ".SILENT") {
		writeLine(w, fmt.Sprintf("* %s FETCH (FLAGS (%s))", msgNum, strings.Join(current, " ")))
	}
	writeLine(w, tag+" OK STORE completed")
}

func handleSearch(w *bufio.Writer, tag string) {
	count := countMails()
	ids := make([]string, count)
	for i := range ids {
		ids[i] = strconv.Itoa(i + 1)
	}
	writeLine(w, "* SEARCH "+strings.Join(ids, " "))
	writeLine(w, tag+" OK SEARCH completed")
}
