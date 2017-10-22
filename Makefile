VERSION=1.5.1
BUILD=`git rev-parse --short HEAD`

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

# Supported platforms
PLATFORMS := linux darwin

temp = $(subst -, ,$@)
os = $(word $(words $(temp)), $(temp))
arch = amd64

dep:
	go get -t ./...

test:
	go test -v ./...

all: dep test release

$(addprefix build-,$(PLATFORMS)):
	mkdir -p binaries/$(os)
	GOOS=$(os) GOARCH=$(arch) go build $(LDFLAGS) -o binaries/$(os)/deploy-ecs cmd/deploy-ecs/main.go

$(addprefix build-static-,$(PLATFORMS)):
	mkdir -p binaries/$(os)
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) go build $(LDFLAGS) -v -a -installsuffix cgo -o binaries/$(os)/deploy-ecs cmd/deploy-ecs/main.go
	tar czf deploy-ecs-$(os).tar.gz -C binaries/$(os) deploy-ecs

$(addprefix release-,$(PLATFORMS)):
	docker run --rm -v `pwd`:/go/src/github.com/guilherme-santos/deploy-ecs golang:latest sh -c "cd /go/src/github.com/guilherme-santos/deploy-ecs && make dep build-static-$(os)"

release: $(addprefix release-,$(PLATFORMS))
