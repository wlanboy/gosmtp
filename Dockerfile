FROM golang:1.15-alpine3.12 as builder
RUN apk add --no-cache git
RUN apk add --no-cache build-base
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go get -d -v
RUN go build -v -o main .

FROM alpine:3.12
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
EXPOSE 25

CMD ["/root/main"]
