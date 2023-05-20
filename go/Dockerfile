FROM golang:alpine

RUN addgroup -S mercari && adduser -S trainee -G mercari
RUN apk update && apk add --no-cache gcc musl-dev sqlite sqlite-dev
# RUN chown -R trainee:mercari /path/to/db

# CMD ["go", "version"]

WORKDIR /app/go
COPY db db
COPY go go

USER root
RUN chown -R trainee:mercari /app/go

USER trainee

ENV GOOS=linux
ENV GOARCH=amd64

# Enable CGO
ENV CGO_ENABLED=1

#ディレクトリ変える
WORKDIR /app/go/go

# Download and cache Go dependencies
RUN go mod download

# Copy the source code into the container
COPY app ./

# Build the Go application
RUN go build -o main

# Set the command to run the Go application
CMD ["./main"]