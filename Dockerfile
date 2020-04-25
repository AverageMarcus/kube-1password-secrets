FROM golang:1.14-alpine AS builder
RUN apk update && apk add --no-cache git curl && adduser -D -g '' gopher && apk add -U --no-cache ca-certificates
RUN curl -o op.zip https://cache.agilebits.com/dist/1P/op/pkg/v0.10.0/op_linux_386_v0.10.0.zip
RUN unzip op.zip && mv ./op /usr/local/bin && chmod +x /usr/local/bin/op
WORKDIR /app/
ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o kube-1password-secrets main.go

FROM golang:1.14-alpine
WORKDIR /app/
COPY --from=builder /usr/local/bin/op /usr/local/bin/op
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/kube-1password-secrets /app/kube-1password-secrets
RUN mkdir -p /root/.op && chmod 700 /root/.op
ENTRYPOINT ["/app/kube-1password-secrets"]
