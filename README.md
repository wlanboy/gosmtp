# gosmtp
golang mock smtp server for local testing - writes sent mail to local directory
- depends on github.com/corpix/smtpd
## imapclient
golang imap client
- depends on github.com/emersion/go-imap
## smtpclient
golang smtp client
- depends on net/smtp and net/mail

# build server
* cd smtp
* go get -d -v
* go build

# build imapclient
* cd smtp/imapclient
* go get -d -v
* go build

# build smtpclient
* cd smtp/smtpclient
* go get -d -v
* go build

# run server
* cd smtp
* go run smtp.go

# run imapclient
* cd smtp/imapclient
* go run client.go

# run smtpclient
* cd smtp/smtpclient
* go run client.go

# debug
* cd smtp
* go get -u github.com/go-delve/delve/cmd/dlv
* dlv debug ./smtp
