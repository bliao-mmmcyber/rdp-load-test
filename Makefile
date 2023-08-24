
ifeq (,$(wildcard mk-common/import.mk))
$(shell git submodule update --init)
endif

IGNORE_DOCKER_DIRS=internal/golang-common
GOCHK_DIRS=cmd lib pkg
DOCKER_BUILD_ARGS=--build-arg guacd=$(DOCKER_REGISTRY)/$(ORG_NAME)/guacd:rel-v23.07.1-00000 ## change once we done with the server $(call get_image_tag)

guac_transcode.iid: DOCKER_REPOSITORY=rdp-transcode
build: docker.build

release: docker.release

test: go.test

include mk-common/import.mk
