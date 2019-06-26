.PHONY: withdrawal-workflow withdrawal-server

# default target
default: withdrawal-workflow withdrawal-server

# Automatically gather all srcs
SRC := $(shell find . -name "*.go")

dep:
	go mod tidy
	go mod vendor

test: dep
	go test -race -v -timeout 5m -coverprofile=test . | tee -a test.log

clean:
	# TODO

withdrawal-server: dep $(SRC)
	go build -i -o dummy-server server/*.go

withdrawal-workflow: dep $(SRC)
	go build -i -o withdrawal *.go


