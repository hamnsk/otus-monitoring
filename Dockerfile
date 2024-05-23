FROM golang:1.21-alpine3.19 as builder

WORKDIR /usr/local/go/src/

COPY ./simple-app/ /usr/local/go/src/

RUN go clean --modcache
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=readonly -o simple-app cmd/server/main.go

FROM scratch
WORKDIR /app
COPY --from=builder /usr/local/go/src/simple-app /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app/simple-app"]
