FROM golang:1.26-bookworm AS builder

RUN apt-get update && \
    apt-get install -y postgresql-common && \
    /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    postgresql-server-dev-15 \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY . .

RUN make prebuild.go && \
    make -C ./fdw go && \
    make -C ./fdw && \
    make -C ./fdw inst

FROM scratch AS artifacts
COPY --from=builder /build/build-Linux/ /
