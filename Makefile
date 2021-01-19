.PHONY: default docker build_tag build build_untag

DOCKEREPO=980993447824.dkr.ecr.us-east-1.amazonaws.com

default:
	@echo only use this makefile to build and push docker image

build_tag:
	-aws --region us-east-1 ecr get-login --no-include-email | sh -
	-aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '$(DOCKEREPO)'

build: DOCKERTAG=appaegis/guac:v1.0
build:
	docker build -t '$(DOCKEREPO)'/'$(DOCKERTAG)' -f Dockerfile --force-rm .
	docker push     '$(DOCKEREPO)'/'$(DOCKERTAG)'

build_untag:

docker: build_tag build build_untag

jenkins-docker: DOCKERTAG=appaegis/guac:$(TAG)
jenkins-docker: LATESTTAG=appaegis/guac:latest
jenkins-docker:
	aws ecr get-login-password --region us-east-1| docker login --username AWS --password-stdin 980993447824.dkr.ecr.us-east-1.amazonaws.com
	docker build --network=host -t '$(DOCKERTAG)' -f Dockerfile --force-rm .
	docker push '$(DOCKERTAG)'
	docker tag $(DOCKERTAG) $(LATESTTAG)
	docker push '$(LATESTTAG)'


