SHELL=/bin/bash -o pipefail

REPO?=embercsi/ember-csi-operator
TAG?="0.0.3"

GOLANG_FILES:=$(shell find . -name \*.go -print)
pkgs = $(shell go list ./... | grep -v /vendor/ )

all: dep compile build 

dep:
	dep ensure -v

clean:
	# Remove all files and directories ignored by git.
	git clean -Xfd .

compile: ember-csi-operator

ember-csi-operator: $(GOLANG_FILES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
	-o build/$@ cmd/manager/main.go

build: compile
	docker build -t $(REPO):$(TAG) -f build/Dockerfile build

push:
	docker push $(REPO):$(TAG)

deploy: 
	oc create -f deploy/install.yaml

undeploy:
	oc delete -f deploy/uninstall.yaml

format: go-fmt

go-fmt:
	go fmt $(pkgs)

.PHONY: dep all clean compile build push deploy undeploy format
