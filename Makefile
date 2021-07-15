.PHONY: default docker build_tag build build_untag

DOCKEREPO=980993447824.dkr.ecr.us-east-1.amazonaws.com
BUILD_BASE=$(DOCKEREPO)/appaegis/golang-builder-base-1.14

default:
	@echo only use this makefile to build and push docker image

build_tag:
	-aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '$(DOCKEREPO)'

build: DOCKERTAG=appaegis/guac:v1.0
build:
	docker pull     '$(BUILD_BASE)'
	docker tag      '$(BUILD_BASE)' build-base
	docker build -t '$(DOCKEREPO)'/'$(DOCKERTAG)' -f Dockerfile --force-rm .
	docker push     '$(DOCKEREPO)'/'$(DOCKERTAG)'

build_untag:

docker: build_tag build build_untag

jenkins-docker: DOCKERTAG='$(DOCKEREPO)'/appaegis/guac:$(TAG)
jenkins-docker: LATESTTAG='$(DOCKEREPO)'/appaegis/guac:latest
jenkins-docker:
	aws ecr get-login-password --region us-east-1| docker login --username AWS --password-stdin 980993447824.dkr.ecr.us-east-1.amazonaws.com
	docker pull     '$(BUILD_BASE)'
	docker tag      '$(BUILD_BASE)' build-base
	docker build --network=host -t '$(DOCKERTAG)' -f Dockerfile --force-rm .
	docker push '$(DOCKERTAG)'

test:
	@go test ./... | grep -v '^?'
