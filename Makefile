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
	goreleaser build --skip-validate --single-target --rm-dist

build_all: fmt scraper
	goreleaser build --skip-validate --rm-dist

release:
	goreleaser release --rm-dist

clean:
	rm -rf dist bin

fmt:
	gofmt -s -l -w .
	terraform fmt --recursive examples

.PHONY: scraper docs
scraper:
	go build -o ./bin/scraper github.com/tivo/terraform-provider-splunk-itsi/scraper

test: fmt
	go test -v -cover github.com/tivo/terraform-provider-splunk-itsi/... -tags test_setup

docs: fmt
	go generate ./...

local_install: build
	rm -rf $(TERRAFORM_PLUGIN_PATH) && mkdir -p $(TERRAFORM_PLUGIN_PATH)
	cp dist/terraform-provider-*/terraform-provider-*  $(TERRAFORM_PLUGIN_PATH)/terraform-provider-splunk-itsi_$(PROVIDER_VERSION)
