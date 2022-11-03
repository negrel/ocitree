VERSION 0.6
FROM docker.io/library/ubuntu:22.04
WORKDIR /usr/src/ocitree

RUN apt update && apt upgrade -y
RUN apt install -y \
	libbtrfs-dev \
	libgpgme-dev \
	libdevmapper-dev \
	ca-certificates \
	git \
	golang-go \
	podman

deps:
	COPY go.mod go.sum .
	RUN go mod download
	COPY cmd/ cmd/
	COPY pkg/ pkg/
	COPY ./*.go .
	RUN go mod tidy

test:
	FROM +deps
	RUN go test -v ./...

build:
	FROM +deps
	RUN go build .
	SAVE ARTIFACT ./ocitree AS LOCAL .

