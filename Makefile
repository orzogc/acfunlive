GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get -d
GITREPO=github.com/orzogc/acfunlive
UIDIR=acfunlive-ui
WINDOWSENV=GOOS=windows GOARCH=amd64
TAGS=-tags tray
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
build: build-go-cli build-ui
build-gui: deps build-go-gui build-ui

build-go-cli:
	$(MKDIR) $(BINARY)
	$(GOBUILD) -o $(BINARY)

build-go-gui:
	$(MKDIR) $(BINARY)
	$(GOBUILD) -o $(BINARY) $(TAGS)

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

build-windows-gui: deps build-go-windows-gui build-ui

build-go-windows-gui:
	$(MKDIR) $(BINARY)
	$(WINDOWSENV) $(GOBUILD) -o $(BINARY) $(TAGS) $(LDFLAGS)

build-windows-cli: deps build-go-windows-cli build-ui

build-go-windows-cli:
	$(MKDIR) $(BINARY)
	$(WINDOWSENV) $(GOBUILD) -o $(BINARY)
