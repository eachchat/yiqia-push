# Build stage
# Use the official Golang v1.16.4-alpine3.13 image as the base image for the build stage
FROM golang:1.22-alpine3.18 AS builder

# Set the working directory to /app
WORKDIR /app
#COPY go.mod go.sum ./
# Download dependencies and verify go.mod file
#RUN go mod download && go mod verify
# Copy everything in the current directory into the container
COPY . .

 # Build the executable with CGO disabled and stripped of symbols
RUN CGO_ENABLED=0 go build -o matrix-push-gateway -mod=vendor -ldflags="-s -w" -installsuffix cgo .

# Runtime stage
# Use the official Alpine 3.13 image as the base image for the runtime stage
FROM alpine:3.17
WORKDIR /workspace
# Copy the built executable into the final image
COPY --from=builder ./app/matrix-push-gateway .
# Set the working directory to /workspace

# Set the default command to run when starting the container
ENTRYPOINT ["./matrix-push-gateway"]
