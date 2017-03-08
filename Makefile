ENV ?= default
CONFIG ?= $(ENV).toml

build: deps config-bindata
	go build

build-release: deps config-bindata
	GOOS=linux GOARCH=amd64 go build

deps:
	go get -d

config-bindata: config
	go-bindata config/

config: config/$(CONFIG)
	cp config/$(CONFIG) config/ec2-hosts.toml

config/$(CONFIG):
	cp config/default.toml.orig config/$(ENV).toml

vet:
	go vet $$(go list ./... | \grep -v /vendor/)

.PHONY: build build-release deps config-bindata config vet
