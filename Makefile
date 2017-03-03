build: deps config-bin-data
	go build

build-release: deps config-bin-data
	GOOS=linux GOARCH=amd64 go build

deps:
	go get -d

config-bin-data:
	go-bindata config/

vet:
	go vet $$(go list ./... | \grep -v /vendor/)
