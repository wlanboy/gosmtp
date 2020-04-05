# gosmtp
golang simple smtp server
- depends on github.com/corpix/smtpd

# build
* go get -d -v
* go clean
* go build

# run
* cd smtp
* go run smtp.go

# debug
* go get -u github.com/go-delve/delve/cmd/dlv
* dlv debug ./smtp
