ENV ?= default

build: deps config-bindata
	go build

build-release: deps config-bindata
	GOOS=linux GOARCH=amd64 go build

deps:
	go get -d

config-bindata: config
	go-bindata config/

config:
	cp config/$(ENV).tml config/ec2-hosts.tml

vet:
	go vet $$(go list ./... | \grep -v /vendor/)

.PHONY: build build-release deps config-bindata config vet
