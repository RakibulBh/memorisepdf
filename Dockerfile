FROM golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12

COPY --from=build /out/api /usr/local/bin/api

# Uploaded lecture files and the log live here, so /data must be a writable
# mount that survives a restart.
WORKDIR /data

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
