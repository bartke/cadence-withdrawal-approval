.PHONY: withdrawal dummy-server approval-system

# default target
default: withdrawal dummy-server approval-system

# Automatically gather all srcs
SRC := $(shell find . -name "*.go")

dep:
	go mod tidy
	go mod vendor

test: dep
	go test -race -v -timeout 5m -coverprofile=test . | tee -a test.log

clean:
	# TODO

dummy-server: dep $(SRC)
	go build -i -o dummy-server server/*.go

approval-system: dep $(SRC)
	go build -i -o auto-approver server/auto-approval-system/*.go

withdrawal: dep $(SRC)
	go build -i -o withdrawal *.go


