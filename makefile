CMD=go build
BINARY_NAME=inventory

# Set up LDFLAGS for 'go build'.  This sets versioning, etc.
VERSION=`cat version.txt`
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION}"

all: buildlinux386 buildlinuxx86_64 buildwin386 buildwinx86_64

buildlinux386:
	@echo ""
	@echo "Building Linux x86."
	GOOS=linux GOARCH=386 ${CMD} ${LDFLAGS} -o bin/${BINARY_NAME}-linux-386 inventory/main.go

buildlinuxx86_64:
	@echo ""
	@echo "Building Linux x86_64."
	GOOS=linux GOARCH=amd64 ${CMD} ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 inventory/main.go

buildwin386:
	@echo ""
	@echo "Building Windows x86."
	GOOS=windows GOARCH=386 ${CMD} ${LDFLAGS} -o bin/${BINARY_NAME}-win-386 inventory/main.go

buildwinx86_64:
	@echo ""
	@echo "Building Windows x86_64."
	GOOS=windows GOARCH=amd64 ${CMD} ${LDFLAGS} -o bin/${BINARY_NAME}-win-amd64 inventory/main.go

clean:
	rm -rfv bin

install:
	@cp -v bin/inventory-linux-amd64 ~/ansible/inventory

fast: clean buildlinuxx86_64 install
