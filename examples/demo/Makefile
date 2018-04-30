ARTIFACT_CONTROLLER=kubervisor-demo

# 0.0 shouldn't clobber any released builds
PREFIX=kubervisor-demo/
#PREFIX = gcr.io/google_containers/

SOURCES := $(shell find $(SOURCEDIR) ! -name "*_test.go" -name '*.go')

CMDBINS := pricer comparator

TAG?=$(shell git tag|tail -1)
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
DATE=$(shell date +%Y-%m-%d/%H:%M:%S )
BUILDINFOPKG=github.com/amadeusitgroup/kubervisor/examples/demo/pkg/utils
LDFLAGS = -ldflags "-w -X ${BUILDINFOPKG}.TAG=${TAG} -X ${BUILDINFOPKG}.COMMIT=${COMMIT} -X ${BUILDINFOPKG}.BRANCH=${BRANCH} -X ${BUILDINFOPKG}.BUILDTIME=${DATE} -s"

all: build

build-%:
	CGO_ENABLED=0 go build -i -installsuffix cgo ${LDFLAGS} -o bin/$* ./cmd/$*

buildlinux-%: ${SOURCES}
	CGO_ENABLED=0 GOOS=linux go build -i -installsuffix cgo ${LDFLAGS} -o docker/$*/$* ./cmd/$*/main.go

container-%: buildlinux-%
	@cd docker/$* && docker build -t $(PREFIX)$*:$(TAG) .

build: $(addprefix build-,$(CMDBINS))

buildlinux: $(addprefix buildlinux-,$(CMDBINS))

container: $(addprefix container-,$(CMDBINS))

test:
	./go.test.sh

push: container
	@cd docker/${ARTIFACT_CONTROLLER} && docker push $(PREFIX)$(ARTIFACT_CONTROLLER):$(TAG)

clean:
	rm -f ${ARTIFACT_CONTROLLER}

# Install all the build and lint dependencies
setup:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	echo "make check" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
.PHONY: setup

# gofmt and goimports all go files
fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
.PHONY: fmt

# Run all the linters
lint:
	gometalinter --vendor ./... -e pkg/client -e _generated -e test --deadline 2m
.PHONY: lint

.PHONY: build push clean test