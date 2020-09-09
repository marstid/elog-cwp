
# Start from golang base image
FROM golang:1.15 as builder

# Add Maintainer Info
LABEL maintainer="MS <marstid@juppi.net>"

# Set the Current Working Directory inside the container
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -trimpath -ldflags "-X main.version=1.0.0 -X main.buildTime=`date +'%Y-%m-%d_%T'`" -o /build/cwp .

######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /build/cwp .

CMD ["./cwp"]