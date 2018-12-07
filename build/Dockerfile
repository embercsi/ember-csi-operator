FROM centos:7

RUN mkdir /etc/ember-csi-operator && chmod 755 /etc/ember-csi-operator
ADD config.yml /etc/ember-csi-operator/config.yml
USER nobody

ADD ember-csi-operator /usr/local/bin/ember-csi-operator

