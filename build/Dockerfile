FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

RUN microdnf update -y && microdnf clean all && rm -rf /var/cache/yum

RUN mkdir /etc/ember-csi-operator && chmod 755 /etc/ember-csi-operator
ADD build/config.yaml /etc/ember-csi-operator/config.yaml
USER nobody

COPY ./ember-csi-operator /usr/local/bin/ember-csi-operator
