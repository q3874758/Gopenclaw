FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o gopenclaw ./cmd/openclaw

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/gopenclaw .
COPY --from=builder /app/ui ./ui
COPY --from=builder /app/internal/config/openclaw.json.example ./config/openclaw.json.example

ENV GOPATH=/go
ENV PATH=$PATH:/app

EXPOSE 11999

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:11999/health || exit 1

ENTRYPOINT ["/app/gopenclaw"]
CMD ["gateway"]
