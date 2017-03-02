build: config-bin-data
	go build

build-release: config-bin-data
	GOOS=linux GOARCH=amd64 go build

config-bin-data:
	go-bindata config/

