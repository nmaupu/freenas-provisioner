FROM alpine

RUN apk update && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/* && \
    update-ca-certificates

COPY bin/freenas-provisioner /freenas-provisioner

ENTRYPOINT ["/freenas-provisioner"]
