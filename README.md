![Go](https://github.com/wlanboy/gosmtp/workflows/Go/badge.svg?branch=master)

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

# run mail mock server
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

# dockerize 
* GOOS=linux GOARCH=386 go build (386 needed for busybox)
* GOOS=linux GOARCH=arm GOARM=6 go build (Raspberry Pi build)
* GOOS=linux GOARCH=arm64 go build (Odroid C2 build)
* docker build -t smtpmock .

# run docker container
* cd smtp
* docker run -d --name smtpmock -p 1025:1025 smtpmock:latest
* docker stop smtpmock && docker rm smtpmock

# test smtp mock server
* sudo apt-get install swaks
* echo "This is the email body" | swaks --to "me@test.com" --from "you@test.com" --server 127.0.0.1 --port 1025
* docker logs smtpmock
