ENV ?= default
CONFIG ?= $(ENV).tml

build: deps config-bindata
	go build

build-release: deps config-bindata
	GOOS=linux GOARCH=amd64 go build

deps:
	go get -d

config-bindata: config
	go-bindata config/

config: config/$(CONFIG)
	cp config/$(CONFIG) config/ec2-hosts.tml

$(CONFIG):
	cp config/default.tml.orig config/$(ENV).tml

vet:
	go vet $$(go list ./... | \grep -v /vendor/)

.PHONY: build build-release deps config-bindata config vet
