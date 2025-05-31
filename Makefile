BINARY_NAME=nfc-monitor
TARGET_ARCH=arm
TARGET_OS=linux
GO=go
GOFLAGS=-ldflags="-s -w"

.PHONY: all build clean

all: build

build:
	GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) GOARM=7 CGO_ENABLED=1 \
		CC=arm-linux-gnueabihf-gcc \
		CXX=arm-linux-gnueabihf-g++ \
		$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

build-local:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME)-local .

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-local

install: build
	@echo "Binary built as $(BINARY_NAME)"
	@echo "Transfer to target device and run with appropriate permissions"