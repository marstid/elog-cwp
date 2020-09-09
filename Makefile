# Go parameters
    GOCMD=go
    GOBUILD=$(GOCMD) build
    GOCLEAN=$(GOCMD) clean
    MODCLEAN=$(GOCMD) clean -modcache
    GOTEST=$(GOCMD) test
    GOGET=$(GOCMD) get
    GOFMT=$(GOCMD) fmt ./...
    BINARY_NAME=cwp
    BINARY_UNIX=$(BINARY_NAME)_unix
    NOW=`date +'%Y-%m-%d_%T'`
    DATE=`date +'%Y-%m-%d'`
    VERSIONGIT=`git rev-parse HEAD`
    VERSION=1.0

    all: test build

    docker: fmt
		docker build . -t $(BINARY_NAME)

    docker-dist:
		rm -rf dist/*.xz
		rm -rf dist/*.tar
		docker save cwp:latest > dist/cwp-$(DATE).tar
		xz -f -9 dist/cwp-$(DATE).tar

    build: fmt
		$(GOBUILD) -o $(BINARY_NAME) -trimpath -ldflags "-X main.sha1ver=$(VERSIONGIT) -X main.buildTime=$(NOW)" -v

    test:
		$(GOTEST) -v ./...

    clean:
		$(GOCLEAN)
		$(MODCLEAN)
		rm -f dist/$(BINARY_NAME)
		rm -f dist/$(BINARY_NAME).tar.xz

    run:
		$(GOBUILD) -o build/ -trimpath -ldflags "-X main.sha1ver=$(VERSIONGIT) -X main.buildTime=$(NOW)" -v ./...
		./build/$(BINARY_NAME)


    # Cross compilation
    dist:
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/$(BINARY_NAME) -ldflags "-X main.sha1ver=$(VERSIONGIT) -X main.buildTime=$(NOW)" -v


	# Format sources
    fmt:
		$(GOFMT)


# Update dependencies
    update:
		go get -u ./...