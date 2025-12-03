# syntax=docker/dockerfile:1

###############################################
# Frontend build stage (mirrors ui-build job) #
###############################################
FROM --platform=$BUILDPLATFORM node:20-alpine AS ui-builder
WORKDIR /src/ui
ENV NODE_ENV=production \
    PNPM_HOME=/usr/local/share/pnpm \
    PATH=/usr/local/share/pnpm:$PATH
RUN apk add --no-cache git \
 && corepack enable \
 && corepack prepare pnpm@latest-9 --activate
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile=false
COPY ui .
RUN pnpm run build-only

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
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /src/ui/dist ./ui/dist
RUN go build -trimpath -ldflags "-s -w" -o /out/sealchat-server .

#############################
# Minimal Alpine runtime    #
#############################
FROM alpine:3.20
WORKDIR /app
RUN addgroup -S sealchat && adduser -S -G sealchat sealchat \
 && apk add --no-cache ca-certificates tzdata ffmpeg \
 && mkdir -p /app/data /app/static /app/temp \
 && chown -R sealchat:sealchat /app
COPY --from=go-builder /out/sealchat-server /usr/local/bin/sealchat-server
COPY config.yaml.example /app/config.yaml.example
EXPOSE 3212
VOLUME ["/app/data", "/app/static", "/app/temp"]
USER sealchat
ENTRYPOINT ["/usr/local/bin/sealchat-server"]
