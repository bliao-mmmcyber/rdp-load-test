.PHONY: default docker build_tag build build_untag

ifndef VERSION
override VERSION=latest
endif

DOCKEREPO=980993447824.dkr.ecr.us-east-1.amazonaws.com
BUILD_BASE=$(DOCKEREPO)/appaegis/golang-builder-base-1.14
GUACD=$(DOCKEREPO)/appaegis/guacd:latest

default:
	@echo only use this makefile to build and push docker image

build_tag:
	-aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '$(DOCKEREPO)'

build: DOCKERTAG=appaegis/guac:$(VERSION)
build:
	docker pull     '$(BUILD_BASE)'
	docker tag      '$(BUILD_BASE)' build-base
	docker build --network=host -t '$(DOCKEREPO)'/'$(DOCKERTAG)' -f Dockerfile --force-rm .
	docker push     '$(DOCKEREPO)'/'$(DOCKERTAG)'

build-transcode: TRANSCODETAG=appaegis/rdp-transcode:$(VERSION)
build-transcode:
	docker pull     '$(BUILD_BASE)'
	docker tag      '$(BUILD_BASE)' build-base
	docker pull     '$(GUACD)'
	docker tag      '$(GUACD)' guacd
	docker build --network=host -t '$(DOCKEREPO)'/'$(TRANSCODETAG)' -f Dockerfile_transcode --force-rm .
	docker push     '$(DOCKEREPO)'/'$(TRANSCODETAG)'

build_untag:

docker: build_tag build build_untag

docker-transcode: build_tag build-transcode

docker-transcoding: build_tag build build_untag

test:
	@go test ./... | grep -v '^?'
