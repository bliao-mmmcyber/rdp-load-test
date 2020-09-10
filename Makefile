.PHONY: default docker build_tag build build_untag

DOCKEREPO=980993447824.dkr.ecr.us-east-1.amazonaws.com

default:
	@echo only use this makefile to build and push docker image

build_tag:
	aws --region=us-east-1 ecr get-login --no-include-email | sh -

build: DOCKERTAG=appaegis/guac:v1.0
build:
	go mod tidy
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./dist/guac  cmd/guac/guac.go
	#
	docker build -t '$(DOCKEREPO)'/'$(DOCKERTAG)' -f Dockerfile --force-rm .
	docker push     '$(DOCKEREPO)'/'$(DOCKERTAG)'

build_untag:

docker: build_tag build build_untag


