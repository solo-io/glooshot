FROM alpine

RUN apk upgrade --update-cache \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

COPY glooshot-op-linux-amd64 /usr/local/bin/glooshot

ENTRYPOINT ["/usr/local/bin/glooshot"]