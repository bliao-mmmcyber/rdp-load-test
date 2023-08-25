ARG guacd
ARG golang_ver

FROM golang:$golang_ver as build-env

ADD . /go/src/app
WORKDIR /go/src/app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN  make go.build

#---------
FROM $guacd

USER root

ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-muslc-amd64 /bin/tini
RUN chmod +x /bin/tini

RUN apt update
RUN apt install -y ffmpeg

COPY --from=build-env /go/src/app/bin/transcoding /home/appaegis/bin/transcoding

RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis
#-------------------

LABEL name="guac" \
        version="1.0"   \
        description="transcoding"

ENTRYPOINT ["/bin/tini", "--"]
CMD ["/home/appaegis/bin/transcoding"]
