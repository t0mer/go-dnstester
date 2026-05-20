# Stage 1: Build the React/Vite UI
FROM node:20-alpine AS ui-builder
WORKDIR /app/web/ui
COPY web/ui/package*.json ./
RUN npm ci
COPY web/ui/ ./
RUN npm run build

# Stage 2: Build the Go binary
FROM golang:1.25-alpine AS go-builder
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui-builder /app/web/ui/dist ./web/dist

RUN GOARM="$(echo "${TARGETVARIANT}" | sed 's/^v//')" && \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GOARM="${GOARM}" \
    go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.buildMode=prod" \
      -o /dnstester \
      ./cmd/dnstester

# Stage 3: Minimal runtime image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=go-builder /dnstester /app/dnstester

ENV CONFIG_PATH=/config
VOLUME ["/config"]
EXPOSE 7020

ENTRYPOINT ["/app/dnstester"]
