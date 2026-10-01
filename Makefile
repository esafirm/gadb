BIN_PATH=$(GOPATH)/bin/gadb

# Support passing positional arguments to make run, make log, and make logs:
# e.g.: make log com.example.app
# e.g.: make logs com.example.app
# e.g.: make run com.example.app
# e.g.: make log ARGS="com.example.app"
ifeq ($(filter $(firstword $(MAKECMDGOALS)),run log logs),$(firstword $(MAKECMDGOALS)))
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(RUN_ARGS):;@:)
endif

.PHONY: all build buildbin run log logs run-dev

all: build
build:
	go build -o gadb -v
buildbin: 
	go build -o $(BIN_PATH)
run: 
	@go run main.go log $(RUN_ARGS) $(ARGS)
log:
	@go run main.go log $(RUN_ARGS) $(ARGS)
logs:
	@go run main.go logs $(RUN_ARGS) $(ARGS)
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
