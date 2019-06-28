
# default target
default: withdraw dummy-server approval-system

# Automatically gather all srcs
SRC := $(shell find . -name "*.go")

dep:
	go mod tidy
	go mod vendor

test:
	go test -race -v -timeout 5m -coverprofile=test . | tee -a test.log

clean:
	# TODO

dummy-server: $(SRC)
	go build -i -o dummy-server server/*.go

approval-system: $(SRC)
	go build -i -o approval-system server/auto-approval-system/*.go

withdraw: $(SRC)
	go build -i -o withdraw *.go


