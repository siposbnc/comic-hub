# ComicHub media server — single static Go binary, pure Go (no cgo), so the runtime
# image can be distroless. Build from the repo root or from server/:
#   docker build -t comichub-server ./server
#
# The server runs in server mode by default here (that's what a container is for);
# enable auth with -e COMICHUB_AUTH_ENABLED=true + the admin bootstrap envs, and mount
# your comics read-only plus a volume for the data dir. See docs/10-deployment.md.

FROM golang:1.26 AS build
WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/comichub-server ./cmd/comichub-server \
    && mkdir /out/data

# Distroless static: no shell, no package manager, tiny attack surface. The server
# needs only its binary, CA certs (metadata providers over HTTPS), and tzdata.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/comichub-server /usr/local/bin/comichub-server
# /data must be writable by the nonroot user; named volumes inherit this ownership.
# (Bind mounts don't — chown the host dir to uid 65532 or run with --user.)
COPY --from=build --chown=nonroot:nonroot /out/data /data

ENV COMICHUB_MODE=server \
    COMICHUB_BIND=0.0.0.0:8080 \
    COMICHUB_DATA_DIR=/data \
    COMICHUB_LOG_FORMAT=json

VOLUME /data
EXPOSE 8080

USER nonroot
ENTRYPOINT ["comichub-server"]
