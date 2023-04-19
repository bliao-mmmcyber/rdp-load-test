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
FROM alpine:latest

ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-muslc-amd64 /bin/tini
RUN chmod +x /bin/tini

# install aws encryption sdk cli
RUN apk add gcc
ENV PYTHONUNBUFFERED=1
RUN apk add --update --no-cache python3 && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip setuptools
RUN apk add python3-dev musl-dev libffi-dev
RUN pip install --upgrade aws-encryption-sdk-cli

COPY --from=build-env /go/src/app/bin/guac /home/appaegis/bin/guac
ADD assets /home/appaegis/guac-assets

ENV ETCD_ENDPOINTS=http://127.0.0.1:2379
ENV ETCD_USERNAME=root
ENV ETCD_PASSWORD=Appaegis1234

RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis
#-------------------

LABEL name="guac" \
        version="1.0"   \
        description="guac"

ENTRYPOINT ["/bin/tini", "--"]
CMD ["/home/appaegis/bin/guac"]
