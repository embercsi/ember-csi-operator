FROM openshift/origin-release:golang-1.10

RUN mkdir -p /go/src/github.com/embercsi/ember-csi-operator/
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

COPY . /go/src/github.com/embercsi/ember-csi-operator/
WORKDIR /go/src/github.com/embercsi/ember-csi-operator/

RUN make dep
RUN make compile

FROM centos:7

RUN yum update -y && yum clean all

RUN mkdir /etc/ember-csi-operator && chmod 755 /etc/ember-csi-operator
ADD build/config.yaml /etc/ember-csi-operator/config.yaml
USER nobody

COPY --from=0 /go/src/github.com/embercsi/ember-csi-operator/build/ember-csi-operator /usr/local/bin/ember-csi-operator
