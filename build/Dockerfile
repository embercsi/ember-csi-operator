FROM centos:7

RUN yum update -y && yum clean all

RUN mkdir /etc/ember-csi-operator && chmod 755 /etc/ember-csi-operator
ADD build/config.yaml /etc/ember-csi-operator/config.yaml
USER nobody

COPY ./ember-csi-operator /usr/local/bin/ember-csi-operator
