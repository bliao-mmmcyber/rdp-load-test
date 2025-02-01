ARG golang_ver GIT_COMMIT
FROM golang:1.23 as build-env

COPY . /go/src/app
WORKDIR /go/src/app

ENV CGO_ENABLED=0 GOOS=linux
RUN make go.build

#---------
FROM 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/golang-common/alpine:rel-4.6.0

ARG GIT_COMMIT
ENV GIT_COMMIT=${GIT_COMMIT}

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
