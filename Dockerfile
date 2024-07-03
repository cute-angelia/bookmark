# Specifies a parent image
# FROM golang:1.20-alpine  as builder
FROM golang:1.22  as builder

# 安装 cgo
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
         gcc libc6-dev

# Creates an app directory to hold your app’s source code
WORKDIR /app
# Copies everything from your root directory into /app
COPY . .
# Installs Go dependencies
RUN go mod tidy

# RUN ls -l ./

# Builds your app with optional configuration
RUN cd cmd/bookmark && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /app/bookmark -ldflags '-extldflags "-static"'
# -- Stage 2 -- #
# Create the final environment with the compiled binary.
FROM alpine
# Install any required dependencies.
# RUN apk --no-cache add ca-certificates
WORKDIR /app
# Copy the binary from the builder stage and set it as the default command.
COPY --from=builder /app/bookmark /app/bookmark

RUN ls -l /app/bookmark

RUN chmod +x /app/bookmark
# Tells Docker which network port your container listens on
EXPOSE 38112
# Specifies the executable command that runs when the container starts
CMD ["/app/bookmark"]