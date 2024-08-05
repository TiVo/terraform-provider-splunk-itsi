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
	go clean -testcache
	rm -rf dist bin

fmt:
	gofmt -s -l -w .
	terraform fmt --recursive examples

.PHONY: scraper docs
scraper:
	go build -o ./bin/scraper github.com/tivo/terraform-provider-splunk-itsi/scraper

# Allows to run a specific test
#
# E.g.:
# make @TestAccResourceServiceKpisLifecycle testacc
# or
# make @'TestAccResourceService.*' testacc
@%:
	$(eval TEST_ARGS := -run $*)

# Run unit test suite
test: fmt
	go test -v -cover -p 1 -parallel=4 $(TEST_ARGS) github.com/tivo/terraform-provider-splunk-itsi/... -tags test_setup

# Run acceptance test suite
testacc: fmt
	TF_ACC=1 TF_ACC_LOG=WARN go test -v -cover -p 1 $(TEST_ARGS) -timeout 60m ./...

# Run sweepers to delete leaked test resources (https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests/sweepers)
sweep: fmt
	TF_ACC_LOG=TRACE go test -v $(TEST_ARGS) -timeout 10m github.com/tivo/terraform-provider-splunk-itsi/provider -sweep=default

docs: fmt
	go generate ./...

local_install: build
	rm -rf $(TERRAFORM_PLUGIN_PATH) && mkdir -p $(TERRAFORM_PLUGIN_PATH)
	cp dist/terraform-provider-*/terraform-provider-*  $(TERRAFORM_PLUGIN_PATH)/terraform-provider-splunk-itsi_$(PROVIDER_VERSION)

update: gomod docs build test

gomod:
	go get -u ./...
	go mod tidy
