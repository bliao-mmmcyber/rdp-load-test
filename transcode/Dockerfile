ARG guacd
ARG golang_ver

FROM golang:1.23 as build-env

ADD . /go/src/app
WORKDIR /go/src/app

ENV CGO_ENABLED=0 GOOS=linux
RUN  make go.build

#---------
FROM $guacd

USER root

RUN apt update
RUN apt install -y ffmpeg

COPY --from=build-env /go/src/app/bin/transcoding /home/appaegis/bin/transcoding

RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis
#-------------------

LABEL name="guac" \
        version="1.0"   \
        description="transcoding"

ENTRYPOINT ["/home/appaegis/bin/transcoding"]
