FROM centos:7

RUN adduser ember-csi-operator
RUN mkdir /etc/ember-csi-operator && chmod 755 /etc/ember-csi-operator
ADD tmp/build/config.yml /etc/ember-csi-operator/config.yml
USER ember-csi-operator

ADD tmp/_output/bin/ember-csi-operator /usr/local/bin/ember-csi-operator
