BIN_PATH=$(GOPATH)/bin/gadb

all: build 
build:
	go build -o gadb -v
buildbin: 
	go build -o $(BIN_PATH)
run: 
	@go run main.go start
run-dev:
	@echo "Select command to run:"
	@select cmd in $$(go run main.go | grep -A 20 "Available Commands:" | awk '/Available Commands:/{f=1;next} /Flags:/{f=0} f && NF{print $$1}'); do \
		if [ -n "$$cmd" ]; then \
			read -p "Enter extra arguments (optional): " args; \
			go run main.go $$cmd $$args; \
			break; \
		else \
			echo "Invalid selection"; \
		fi; \
	done
