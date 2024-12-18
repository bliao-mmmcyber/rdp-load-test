
ifeq (,$(wildcard mk-common/import.mk))
$(shell git submodule update --init)
endif

IGNORE_DOCKER_DIRS=internal/golang-common
GOCHK_DIRS=cmd lib pkg
DOCKER_BUILD_ARGS=--build-arg guacd=$(DOCKER_REGISTRY)/$(ORG_NAME)/guacd:dev-b0f7d08 ## change once we done with the server $(call get_image_tag)

guac_transcode.iid: DOCKER_REPOSITORY=rdp-transcode
build: docker.build

release: docker.release

test: go.test

include mk-common/import.mk
