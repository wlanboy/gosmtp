package main

import (
	"crypto/tls"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func main() {

	imapserver := "mailserver"
	username := ""
	userpassowrd := ""

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         imapserver,
	}

	imapclient, err := client.DialTLS(imapserver+":993", tlsconfig)
	if err != nil {
		log.Fatal(err)
	}
	defer imapclient.Logout()

	if err := imapclient.Login(username, userpassowrd); err != nil {
		log.Fatal(err)
	}

	mailfolders := make(chan *imap.MailboxInfo, 10)

	imapclient.List("", "*", mailfolders)

	log.Println("folders")
	for m := range mailfolders {
		log.Println("* " + m.Name)
	}

	mbox, err := imapclient.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("flags", mbox.Flags)

	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 10 {
		from = mbox.Messages - 10
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	imapclient.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)

	log.Println("messages")
	for msg := range messages {
		log.Printf("* %s - %s - %s", msg.Envelope.Subject, msg.Envelope.From[0].Address(), msg.Envelope.To[0].Address())
	}
}
