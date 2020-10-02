# build stage
FROM golang as builder

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY requestHeadersQueryParamsAndBody.go main.go

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go

# final stage
FROM scratch
COPY --from=builder /app/main /app/main
EXPOSE 5464
ENTRYPOINT ["/app/main"]
