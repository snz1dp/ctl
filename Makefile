export SHELL:=/bin/bash

OS:=$(shell uname | awk '{print tolower($$0)}')

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TARGET_NAME=snz1dpctl
TARGET_VERSION=$(shell cat $(ROOT_DIR)/VERSION)
LDFLAGS="-s -w"
COMMIT_ID:=$(shell git rev-parse HEAD)
DOCKER_CMD:=$(shell if [ -x "$(command -v podman)" ]; then echo podman; else echo docker; fi)

export GOPATH
export PATH

GOVERSION=$(shell go version)

.PHONY: debug

debug:
	@echo OS=$(OS)
	@echo PATH=$(PATH)
	@echo ROOT_DIR=$(ROOT_DIR)
	@echo TARGET_NAME=$(TARGET_NAME)
	@echo TARGET_VERSION=$(TARGET_VERSION)
	@echo GOVERSION=$(GOVERSION)
	@echo GOPATH=$(GOPATH)
	@echo CONTAINER=$(DOCKER_CMD)
	@echo make package-asset
	@echo make build
	@echo make depends

.PHONY: submodule
submodule:
	git submodule update --init --recursive

.PHONY: depends

depends: submodule
	go get github.com/GeertJohan/go.rice/rice
	go get

.PHONY: asset

copy-version:
	mkdir -p $(ROOT_DIR)/asset/version
	cp -f $(ROOT_DIR)/VERSION $(ROOT_DIR)/asset/version
	echo -n $(COMMIT_ID)>$(ROOT_DIR)/asset/version/COMMITID


package-asset: copy-version
	cd action && $(GOPATH)/bin/rice embed-go
	cd utils && $(GOPATH)/bin/rice embed-go

.PHONY: build

out/$(TARGET_NAME)-linux-amd64:
	mkdir -p $(ROOT_DIR)/out
	GOPROXY=https://goproxy.cn GO111MODULE=on GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(ROOT_DIR)/out/$(TARGET_NAME)-linux-amd64
	sha256sum $(ROOT_DIR)/out/$(TARGET_NAME)-linux-amd64>$(ROOT_DIR)/out/$(TARGET_NAME)-linux-amd64.sha256

out/$(TARGET_NAME)-linux-arm64:
	mkdir -p $(ROOT_DIR)/out
	GOPROXY=https://goproxy.cn GO111MODULE=on GOOS=linux GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(ROOT_DIR)/out/$(TARGET_NAME)-linux-arm64
	sha256sum $(ROOT_DIR)/out/$(TARGET_NAME)-linux-arm64>$(ROOT_DIR)/out/$(TARGET_NAME)-linux-arm64.sha256

out/$(TARGET_NAME)-darwin-amd64:
	mkdir -p $(ROOT_DIR)/out
	GOPROXY=https://goproxy.cn GO111MODULE=on GOOS=darwin GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(ROOT_DIR)/out/$(TARGET_NAME)-darwin-amd64
	sha256sum $(ROOT_DIR)/out/$(TARGET_NAME)-darwin-amd64>$(ROOT_DIR)/out/$(TARGET_NAME)-darwin-amd64.sha256

out/$(TARGET_NAME)-darwin-arm64:
	mkdir -p $(ROOT_DIR)/out
	GOPROXY=https://goproxy.cn GO111MODULE=on GOOS=darwin GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(ROOT_DIR)/out/$(TARGET_NAME)-darwin-arm64
	sha256sum $(ROOT_DIR)/out/$(TARGET_NAME)-darwin-arm64>$(ROOT_DIR)/out/$(TARGET_NAME)-darwin-arm64.sha256

out/$(TARGET_NAME)-windows-amd64.exe:
	mkdir -p $(ROOT_DIR)/out
	GOPROXY=https://goproxy.cn GO111MODULE=on GOOS=windows GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(ROOT_DIR)/out/$(TARGET_NAME)-windows-amd64.exe
	sha256sum $(ROOT_DIR)/out/$(TARGET_NAME)-windows-amd64.exe>$(ROOT_DIR)/out/$(TARGET_NAME)-windows-amd64.exe.sha256

out/$(TARGET_NAME)-windows-arm64.exe:
	mkdir -p $(ROOT_DIR)/out
	GOPROXY=https://goproxy.cn GO111MODULE=on GOOS=windows GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(ROOT_DIR)/out/$(TARGET_NAME)-windows-arm64.exe
	sha256sum $(ROOT_DIR)/out/$(TARGET_NAME)-windows-arm64.exe>$(ROOT_DIR)/out/$(TARGET_NAME)-windows-arm64.exe.sha256

build-release:
	mkdir -p $(ROOT_DIR)/out
	$(MAKE) out/$(TARGET_NAME)-linux-amd64
	$(MAKE) out/$(TARGET_NAME)-linux-arm64
	$(MAKE) out/$(TARGET_NAME)-darwin-amd64
	$(MAKE) out/$(TARGET_NAME)-darwin-arm64
	$(MAKE) out/$(TARGET_NAME)-windows-amd64.exe

docker: build
	snz1dpctl make docker

package:
	snz1dpctl make package

build: package-asset
	$(DOCKER_CMD) run --rm -v $(ROOT_DIR):/ctl \
		-e CGO_ENABLED=0 \
		-e GODEBUG="invalidcpu=ignore" \
		snz1.cn/dp/golang:1.25.0 \
		bash -c "cd /ctl && make build-release";

all: clean build

clean:
	rm -rf $(ROOT_DIR)/out/*

publish: clean build
	snz1dpctl make publish
	scp -P 92 $(ROOT_DIR)/out/* root@gitlab.snz1.cn:/data/download/snz1dp/
	scp -P 92 $(ROOT_DIR)/asset/version/* root@gitlab.snz1.cn:/data/download/snz1dp/
