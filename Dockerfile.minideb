FROM bitnami/minideb:stretch

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y ca-certificates && \
    update-ca-certificates && \
    apt-get clean

COPY tmp/freenas-provisioner /
ENTRYPOINT ["/freenas-provisioner"]
