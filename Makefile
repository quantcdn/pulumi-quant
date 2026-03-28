.PHONY: tfgen provider build generate generate_nodejs generate_python generate_go generate_dotnet clean

PACK := quant
ORG := quantcdn
PROJECT := github.com/$(ORG)/pulumi-$(PACK)
VERSION_PATH := provider/pkg/version.Version
PROVIDER_VERSION ?= 0.1.0

LDFLAGS := -X $(PROJECT)/$(VERSION_PATH)=$(PROVIDER_VERSION)

build: tfgen provider

tfgen:
	cd provider && go build -ldflags "$(LDFLAGS)" -o ../bin/pulumi-tfgen-$(PACK) ./cmd/pulumi-tfgen-$(PACK)

provider: tfgen
	cd provider && go build -ldflags "$(LDFLAGS)" -o ../bin/pulumi-resource-$(PACK) ./cmd/pulumi-resource-$(PACK)

generate: tfgen generate_schema generate_nodejs generate_python generate_go generate_dotnet

generate_schema: tfgen
	./bin/pulumi-tfgen-$(PACK) schema --out provider/cmd/pulumi-resource-$(PACK)

generate_nodejs: tfgen
	./bin/pulumi-tfgen-$(PACK) nodejs --out sdk/nodejs

generate_python: tfgen
	./bin/pulumi-tfgen-$(PACK) python --out sdk/python

generate_go: tfgen
	./bin/pulumi-tfgen-$(PACK) go --out sdk/go

generate_dotnet: tfgen
	./bin/pulumi-tfgen-$(PACK) dotnet --out sdk/dotnet

build_nodejs: generate_nodejs
	cd sdk/nodejs && yarn install && yarn run build

install_nodejs_sdk: build_nodejs
	cd sdk/nodejs && yarn link --silent

clean:
	rm -rf bin/ sdk/
