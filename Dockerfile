ARG VERSION=dev

# Build stage
FROM golang:1.22-alpine AS builder
ARG VERSION
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /agentroom \
    ./cmd/agentroom

# Final stage — minimal image
FROM scratch
COPY --from=builder /agentroom /agentroom
VOLUME ["/data"]
EXPOSE 8080
ENV AGENTROOM_DB=/data/agentroom.db
ENV AGENTROOM_BIND=0.0.0.0
ENTRYPOINT ["/agentroom"]
