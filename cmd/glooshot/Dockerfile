FROM alpine

RUN apk upgrade --update-cache \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

COPY glooshot-linux-amd64 /usr/local/bin/glooshot

ENTRYPOINT ["/usr/local/bin/apiserver"]