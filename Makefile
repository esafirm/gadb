BIN_PATH=$(GOPATH)/bin/gadb

all: build 
build:
	go build -o gadb -v
buildbin: 
	go build -o $(BIN_PATH)
run: 
	@go run main.go install ~/Desktop/bl-1081.apk
run-dev:
	@go run main.go avds
