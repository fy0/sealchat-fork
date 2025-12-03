# syntax=docker/dockerfile:1

###############################################
# Frontend build stage (mirrors ui-build job) #
###############################################
FROM --platform=$BUILDPLATFORM node:20-alpine AS ui-builder
WORKDIR /src/ui
ENV NODE_ENV=production \
    YARN_CACHE_FOLDER=/tmp/.yarn-cache
RUN apk add --no-cache git
COPY ui/package.json ui/yarn.lock ./
RUN corepack enable \
 && corepack prepare yarn@stable --activate \
 && yarn install --frozen-lockfile
COPY ui .
RUN yarn build-only

################################################
# Backend build stage (mirrors backend-build)  #
################################################
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS go-builder
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    CGO_ENABLED=0
RUN apk add --no-cache build-base pkgconf git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /src/ui/dist ./ui/dist
RUN go build -trimpath -ldflags "-s -w" -o /out/sealchat-server .

################################################
# Prepare Alpine rootfs for target architecture #
################################################
FROM --platform=$BUILDPLATFORM alpine:3.20 AS runtime-prep
ARG TARGETARCH
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) APK_ARCH="x86_64" ;; \
      arm64) APK_ARCH="aarch64" ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    mkdir -p /rootfs/etc/apk; \
    cp -r /etc/apk/keys /rootfs/etc/apk/; \
    echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/main" > /rootfs/etc/apk/repositories; \
    echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/community" >> /rootfs/etc/apk/repositories; \
    apk --root /rootfs --arch "${APK_ARCH}" --update-cache --initdb add \
        alpine-base ca-certificates tzdata ffmpeg; \
    rm -rf /rootfs/var/cache/apk/*; \
    install -d /rootfs/app/data /rootfs/app/static /rootfs/app/temp
RUN set -eux; \
    if ! grep -q '^sealchat:' /rootfs/etc/group; then \
      echo 'sealchat:x:10001:' >> /rootfs/etc/group; \
    fi; \
    if ! grep -q '^sealchat:' /rootfs/etc/passwd; then \
      echo 'sealchat:x:10001:10001:SealChat:/app:/sbin/nologin' >> /rootfs/etc/passwd; \
    fi; \
    chown -R 10001:10001 /rootfs/app

#############################
# Minimal Alpine runtime    #
#############################
FROM --platform=$TARGETPLATFORM scratch
WORKDIR /app
COPY --from=runtime-prep /rootfs/ /
COPY --from=go-builder /out/sealchat-server /usr/local/bin/sealchat-server
COPY config.yaml.example /app/config.yaml.example
EXPOSE 3212
VOLUME ["/app/data", "/app/static", "/app/temp"]
USER sealchat
ENTRYPOINT ["/usr/local/bin/sealchat-server"]
