# gosmtp
Golang mock SMTP and IMAP server for local testing — writes sent mails to a local `mails/` directory.

## Project structure

```
server/         — SMTP + IMAP server (server.go, smtp.go, imap.go)
imapclient/     — example IMAP client smtpclient/     — example SMTP client (uses net/smtp)
```

## Ports

| Protocol | Default address  |
|----------|-----------------|
| SMTP     | 127.0.0.1:1025  |
| IMAP     | 127.0.0.1:1143  |

Default credentials: `testuser` / `testpass`

---

## Build & run server

```bash
cd server
go build -o gosmtp .
./gosmtp
```

Or without building:

```bash
cd server
go run .
```

## Test server

```bash
cd server
go test ./...
```

## Debug server

```bash
cd server
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug .
```

---

## Build & run imapclient

```bash
cd imapclient
go build .
./imapclient
# or:
go run client.go
```

## Build & run smtpclient

```bash
cd smtpclient
go build .
./smtpclient
# or:
go run client.go
```

---

## Docker

Build:

```bash
cd server
docker build -t smtpmock .
```

Run:

```bash
docker run --rm  --name smtpmock -p 1025:1025 -p 1143:1143 smtpmock:latest
docker run -d    --name smtpmock -p 1025:1025 -p 1143:1143 smtpmock:latest
docker stop smtpmock && docker rm smtpmock
```

Cross-compile for Docker targets:

```bash
GOOS=linux GOARCH=amd64  go build -o gosmtp .
GOOS=linux GOARCH=arm64  go build -o gosmtp .   # Raspberry Pi 4 / Odroid C2
GOOS=linux GOARCH=arm GOARM=6 go build -o gosmtp .  # Raspberry Pi 1/2/3
```

---

## Manual SMTP test with swaks

```bash
sudo apt-get install swaks

swaks \
  --to "me@test.com" \
  --from "you@test.com" \
  --server 127.0.0.1 \
  --port 1025 \
  --auth-user testuser \
  --auth-password testpass \
  --body "This is the email body"

docker logs smtpmock
```
