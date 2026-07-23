all: build-cosmovisor-linux

COSMOVISOR_IMAGE_TAG ?= local

build-cosmovisor-linux:
	mkdir -p $(BUILDDIR)
	docker build --platform linux/amd64 --no-cache --tag dochain/dochaind.cosmovisor-binary:$(COSMOVISOR_IMAGE_TAG) --file Dockerfile.cosmovisor .
	docker create --platform linux/amd64 --name temp dochain/dochaind.cosmovisor-binary:$(COSMOVISOR_IMAGE_TAG)
	docker cp temp:/go/bin/cosmovisor $(BUILDDIR)/
	docker rm temp

.PHONY: all build-cosmovisor-linux





