PROVIDER_VERSION := 99.0.0
TERRAFORM_PLUGIN_PATH := local/itsi/splunk-itsi/$(PROVIDER_VERSION)/
OS := $(shell uname| tr '[:upper:]' '[:lower:]')
ARCH := $(shell arch)

ifeq ($(OS), linux)
	TERRAFORM_PLUGIN_PATH := ~/.terraform.d/plugins/$(TERRAFORM_PLUGIN_PATH)$(OS)_$(ARCH)
endif
ifeq ($(OS), darwin)
	TERRAFORM_PLUGIN_PATH := ~/Library/Application\ Support/io.terraform/plugins/$(TERRAFORM_PLUGIN_PATH)$(OS)_$(ARCH)
endif

default: build

build: fmt scraper
	goreleaser build --single-target --snapshot --clean

build_all: fmt scraper
	goreleaser build --snapshot --clean

release:
	goreleaser release --clean

clean:
	rm -rf dist bin

fmt:
	gofmt -s -l -w .
	terraform fmt --recursive examples

.PHONY: scraper docs
scraper:
	go build -o ./bin/scraper github.com/tivo/terraform-provider-splunk-itsi/scraper

@%:
	$(eval TEST_ARGS := -run $*)

test: fmt
	go test -v -cover -parallel=4 $(TEST_ARGS) github.com/tivo/terraform-provider-splunk-itsi/... -tags test_setup

testacc: fmt
	go clean -testcache
	TF_ACC=1 go test -v -cover $(TEST_ARGS) -timeout 60m ./...

sweep: fmt
	TF_ACC_LOG=trace go test -v $(TEST_ARGS) -timeout 10m github.com/tivo/terraform-provider-splunk-itsi/provider -sweep=default

docs: fmt
	go generate ./...

local_install: build
	rm -rf $(TERRAFORM_PLUGIN_PATH) && mkdir -p $(TERRAFORM_PLUGIN_PATH)
	cp dist/terraform-provider-*/terraform-provider-*  $(TERRAFORM_PLUGIN_PATH)/terraform-provider-splunk-itsi_$(PROVIDER_VERSION)

update: gomod docs build test

gomod:
	go get -u ./...
	go mod tidy
