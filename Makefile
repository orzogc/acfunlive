GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get -d
WINDOWSENV=GOOS=windows GOARCH=amd64
LDFLAGS=-ldflags -H=windowsgui
MKDIR=mkdir -p
RM=rm -rf
BINARY=bin

all: build

build:
	$(MKDIR) $(BINARY)
ifeq ($(OS),Windows_NT)
	$(WINDOWSENV) $(GOBUILD) -o $(BINARY) $(LDFLAGS)
else
	$(GOBUILD) -o $(BINARY)
endif

clean:
	$(GOCLEAN)
	$(RM) $(BINARY)
