all: dochaind-env dochaind-upgrade-env

dochaind-env:
	docker build --platform linux/amd64 --no-cache --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --tag daviddochain/dochaind-env terrad-env

dochaind-upgrade-env:
	docker build --platform linux/amd64 --no-cache --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --build-arg entrypoint=cosmovisor-entrypoint.sh --tag daviddochain/dochaind-upgrade-env terrad-env

.PHONY: all dochaind-env dochaind-upgrade-env






