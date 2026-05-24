package main

import (
	"bufio"
	"encoding/base64"
	"net"
	"strings"
)

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
		case dataMode:
			if cmd == "." {
				saveMail(from, to, data.String())
				data.Reset()
				dataMode = false
				from = ""
				to = nil
				writeLine(w, "250 Message accepted")
			} else {
				data.WriteString(strings.TrimPrefix(cmd, ".") + "\n") // dot-stuffing per RFC 5321
			}

		case strings.HasPrefix(upper, "EHLO"):
			writeLine(w, "250-localhost greets you")
			writeLine(w, "250-AUTH LOGIN PLAIN")
			writeLine(w, "250 OK")

		case strings.HasPrefix(upper, "HELO"):
			writeLine(w, "250 localhost")

		case upper == "NOOP":
			writeLine(w, "250 OK")

		case upper == "RSET":
			from = ""
			to = nil
			data.Reset()
			dataMode = false
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
			from = cmd[len("MAIL FROM:"):]
			to = nil
			writeLine(w, "250 OK")

		case strings.HasPrefix(upper, "RCPT TO:"):
			if !auth {
				writeLine(w, "530 Authentication required")
				continue
			}
			to = append(to, cmd[len("RCPT TO:"):])
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

		case upper == "QUIT":
			writeLine(w, "221 Bye")
			return

		default:
			writeLine(w, "502 Command not implemented")
		}
	}
}
