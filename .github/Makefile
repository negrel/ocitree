.PHONY: deps
deps:
	sudo apt install -y \
		libbtrfs-dev \
		libgpgme-dev \
		libdevmapper-dev \
		ca-certificates \
		git \
		golang-go \
		podman

.PHONY: push
push: deps
	$(MAKE) -C .. test
