ENV ?= default
CONFIG ?= $(ENV).toml

build: deps
	go build

build-release: deps
	GOOS=linux GOARCH=amd64 go build

deps:
	go get -d

config: config/$(CONFIG)
	cp config/$(CONFIG) config/ec2-hosts.toml

config/$(CONFIG):
	cp config/default.toml.orig config/$(ENV).toml

vet:
	go vet $$(go list ./... | \grep -v /vendor/)

.PHONY: build build-release deps config vet
