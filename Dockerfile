# Build
FROM golang:1.20-alpine AS build

# Install dependencies
RUN apk update && apk upgrade && apk add --no-cache \
  make git

WORKDIR /app

COPY . .

RUN make build-linux

# Final container
FROM alpine:3.18

WORKDIR /app

COPY --from=build /app/bin/linux/wp-go-static /app/

RUN chmod u+x /app/wp-go-static

# Start
ENTRYPOINT [ "/app/wp-go-static" ]