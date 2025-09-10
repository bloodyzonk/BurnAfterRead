# Build stage
#FROM golang:1.25 AS builder
FROM alpine:latest AS builder

RUN apk add --no-cache build-base sqlite-dev go

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o /BurnAfterRead .

# Final minimal image
FROM alpine:latest

RUN apk add --no-cache sqlite-libs

# Create a user and group with specific IDs
RUN addgroup -g 1000 bar && \
  adduser -D -u 1000 -G bar bar

USER bar

WORKDIR /app

COPY --from=builder /BurnAfterRead /BurnAfterRead

EXPOSE 8080
ENTRYPOINT ["/BurnAfterRead"]
