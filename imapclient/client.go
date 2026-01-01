package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func main() {
	imapserver := "127.0.0.1:1143"
	username := "testuser"
	userpassword := "testpass"

	// 1. Verbindung zum Server herstellen (PLAIN)
	c, err := client.Dial(imapserver)
	if err != nil {
		log.Fatal("IMAP connect failed:", err)
	}
	defer c.Logout()

	log.Println("Connected to IMAP server")

	// 2. Login
	if err := c.Login(username, userpassword); err != nil {
		log.Fatal("IMAP login failed:", err)
	}
	log.Println("Logged in")

	// 3. Ordner auflisten
	mailfolders := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailfolders)
	}()

	log.Println("Verfügbare Ordner:")
	for m := range mailfolders {
		log.Println("-", m.Name)
	}
	if err := <-done; err != nil {
		log.Fatal("LIST fehlgeschlagen:", err)
	}

	// 4. INBOX auswählen
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal("SELECT fehlgeschlagen:", err)
	}
	log.Printf("INBOX enthält %d Nachrichten.", mbox.Messages)

	if mbox.Messages == 0 {
		log.Println("Keine Nachrichten zum Abrufen vorhanden.")
		return
	}

	// 5. Bereich festlegen (letzte 10 Nachrichten)
	from := uint32(1)
	if mbox.Messages > 10 {
		from = mbox.Messages - 9
	}
	to := mbox.Messages
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	// 6. FETCH Items definieren
	// Wir wollen den Envelope (Metadaten) und BODY[] (Inhalt)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	log.Println("\n--- Nachrichtendetails ---")
	for msg := range messages {
		if msg == nil {
			continue
		}

		fmt.Printf("\n[Seq-ID: %d] [UID: %d]\n", msg.SeqNum, msg.Uid)

		// ENVELOPE (Header-Daten)
		if msg.Envelope != nil {
			fmt.Printf("Datum:   %s\n", msg.Envelope.Date.Format("02.01.2006 15:04"))
			fmt.Printf("Von:     %s\n", formatAddress(msg.Envelope.From))
			fmt.Printf("Betreff: %s\n", msg.Envelope.Subject)
		}

		// BODY (Inhalt)
		r := msg.GetBody(section)
		if r != nil {
			bodyBytes, err := io.ReadAll(r)
			if err != nil {
				log.Println("Fehler beim Lesen des Bodys:", err)
			} else {
				fmt.Println("Inhalt:")
				fmt.Println(strings.Repeat("-", 20))
				fmt.Println(string(bodyBytes))
				fmt.Println(strings.Repeat("-", 20))
			}
		} else {
			fmt.Println("Inhalt: (Nicht mitgeliefert)")
		}
	}

	if err := <-done; err != nil {
		log.Fatal("FETCH abgeschlossen mit Fehler:", err)
	}

	log.Println("\nAbruf beendet.")
}

// Hilfsfunktion zur Darstellung von IMAP-Adressen
func formatAddress(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return "Unbekannt"
	}
	var res []string
	for _, a := range addrs {
		res = append(res, fmt.Sprintf("%s <%s@%s>", a.PersonalName, a.MailboxName, a.HostName))
	}
	return strings.Join(res, ", ")
}
