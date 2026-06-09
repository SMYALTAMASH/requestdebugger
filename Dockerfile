# build stage
FROM golang as builder

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod main.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# final stage
FROM scratch
COPY --from=builder /app/main /app/main
EXPOSE 5464
STOPSIGNAL SIGTERM
ENTRYPOINT ["/app/main"]
