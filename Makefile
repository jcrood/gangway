# Copyright © 2017 Heptio
# Copyright © 2017 Craig Tracey
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PROJECT := gangway
# Where to push the docker image.
REGISTRY ?= jcrood
IMAGE := $(REGISTRY)/$(PROJECT)
SRCDIRS := ./cmd/gangway
PKGS := $(shell go list ./cmd/... ./internal/...)

VERSION ?= master

all: build

build: deps
	go build ./...

install:
	go install -v ./cmd/gangway/...

check: test vet gofmt staticcheck misspell

deps:
	go mod tidy && go mod vendor && go mod verify

vet: | test
	go vet ./...

test:
	go test -v ./...

staticcheck:
	@go get honnef.co/go/tools/cmd/staticcheck
	staticcheck $(PKGS)

misspell:
	@go get github.com/client9/misspell/cmd/misspell
	misspell \
		-i clas \
		-locale US \
		-error \
		cmd/* docs/* *.md

gofmt:
	@echo Checking code is gofmted
	@test -z "$(shell gofmt -s -l -d -e $(SRCDIRS) | tee /dev/stderr)"

image:
	docker build . -t $(IMAGE):$(VERSION)

push:
	docker push $(IMAGE):$(VERSION)

.PHONY: all deps test image
