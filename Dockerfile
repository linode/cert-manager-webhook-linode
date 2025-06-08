FROM alpine:3.18

LABEL org.opencontainers.image.source=http://github.com/nalum/cert-manager-webhook-linode

RUN apk add --no-cache ca-certificates

COPY webhook /webhook

ENTRYPOINT ["/webhook"]
