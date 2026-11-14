FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG BINARY=public-gateway
RUN CGO_ENABLED=0 go build -o /app ./cmd/${BINARY}
FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D doodhee
USER doodhee
COPY --from=build /app /app
ENTRYPOINT ["/app"]
