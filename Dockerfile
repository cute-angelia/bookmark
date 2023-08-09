# Specifies a parent image
FROM golang:1.20-alpine  as builder
# Creates an app directory to hold your app’s source code
WORKDIR /app
# Copies everything from your root directory into /app
COPY . .
# Installs Go dependencies
RUN go mod download

# Builds your app with optional configuration
RUN cd cmd/bookmark && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bookmarkapi -ldflags '-extldflags "-static"'
# -- Stage 2 -- #
# Create the final environment with the compiled binary.
FROM alpine
# Install any required dependencies.
# RUN apk --no-cache add ca-certificates
WORKDIR /app
# Copy the binary from the builder stage and set it as the default command.
COPY --from=builder /app/bookmarkapi /app/bookmarkapi

RUN ls -l /app/bookmarkapi

RUN chmod +x /app/bookmarkapi
# Tells Docker which network port your container listens on
EXPOSE 38112
# Specifies the executable command that runs when the container starts
CMD ["/app/bookmarkapi"]