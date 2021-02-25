#!/bin/bash
# The Python script accepts CONSOLE_VERSION and DEVELOPMENT env variables
TAG="${1:-master}"

if [[ -z "${DEVELOPMENT}" ]]; then
  dest=../../deploy/olm-catalog/next
  dest_file=${2:-$dest/ember-csi-operator.vX.Y.Z.clusterserviceversion.yaml}
  mkdir -p $dest
else
  dest_file=./out.yaml
fi

podman pull embercsi/ember-csi:$TAG

echo "Getting driver config from tag $TAG and writing result to $dest_file"
docker run --rm embercsi/ember-csi:$TAG ember-list-drivers -d | python ./yaml-options-gen.py > $dest_file
