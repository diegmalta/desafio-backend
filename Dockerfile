# Build
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/server

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/server /server
EXPOSE 8080
USER nobody
ENTRYPOINT ["/server"]
