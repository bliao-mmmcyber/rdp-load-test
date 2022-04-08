
ifeq (,$(wildcard mk-common/import.mk))
$(shell git submodule update --init)
endif
DOCKER_BUILD_ARGS=--build-arg guacd=$(DOCKER_REGISTRY)/$(ORG_NAME)/guacd:latest ## change once we done with the server $(call get_image_tag)

build: docker.build

release: docker.release

test: go.test

include mk-common/import.mk
