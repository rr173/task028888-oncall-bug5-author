# syntax=docker/dockerfile:1

# Builder: matches the local Go toolchain exactly. Vendored deps + stdlib only,
# so the build runs fully offline once the source is copied in.
FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm AS builder
WORKDIR /src
ENV GOTOOLCHAIN=local \
    GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn \
    CGO_ENABLED=0 \
    GOFLAGS=-mod=vendor
COPY . .
RUN go build -o /out/oncall .

# Runtime: static binary on a minimal image, no external services needed.
FROM docker.m.daocloud.io/library/alpine:3.20
COPY --from=builder /out/oncall /usr/local/bin/oncall
ENTRYPOINT ["/usr/local/bin/oncall"]
