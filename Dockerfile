ARG golang_builder_base
ARG golang_ver
FROM $golang_builder_base as base
FROM golang:$golang_ver as build-env

WORKDIR /go/src/app
COPY --from=base /go/src/golang-common /go/src/golang-common
ADD . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN make go.build

#---------
FROM 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/golang-common/alpine

COPY --from=build-env /go/src/app/bin/guac /home/appaegis/bin/guac
ADD assets /home/appaegis/guac-assets

RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis
#-------------------

LABEL name="guac" \
        version="1.0"   \
        description="guac"

ENTRYPOINT ["/bin/tini", "--"]
CMD ["/home/appaegis/bin/guac"]
