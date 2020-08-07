GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get -d
GITREPO=github.com/orzogc/acfunlive
UIDIR=acfunlive-ui
WINDOWSENV=GOOS=windows GOARCH=amd64
LDFLAGS=-ldflags -H=windowsgui
YARNINSTALL=yarn install
YARNGENERATE=yarn generate
NODEMODULES=node_modules
NUXTDIR=.nuxt
DISTDIR=dist
MKDIR=mkdir -p
RM=rm -rf
CP=cp -rf
CD=cd
BINARY=bin
WEBUIDIR=webui

all: deps build
build: build-go build-ui

build-go:
	$(MKDIR) $(BINARY)
	$(GOBUILD) -o $(BINARY)

build-ui:
	$(CD) $(UIDIR) && $(YARNGENERATE)
	$(CP) $(UIDIR)/$(DISTDIR)/. $(BINARY)/$(WEBUIDIR)

deps:
	$(GOGET) $(GITREPO)
	$(CD) $(UIDIR) && $(YARNINSTALL)

clean:
	$(GOCLEAN)
	$(RM) $(BINARY)
	$(RM) $(UIDIR)/$(NODEMODULES)
	$(RM) $(UIDIR)/$(NUXTDIR)
	$(RM) $(UIDIR)/$(DISTDIR)

build-go-windows:
	$(MKDIR) $(BINARY)
	$(WINDOWSENV) $(GOBUILD) -o $(BINARY) $(LDFLAGS)

build-windows: deps build-go-windows build-ui
