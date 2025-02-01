FROM golang:1.23 as build-env

ADD . /go/src/app
WORKDIR /go/src/app

ENV CGO_ENABLED=0 GOOS=linux
RUN make go.build

#---------
FROM alpine:latest

COPY --from=build-env /go/src/app/bin/stresstest /home/appaegis/bin/stresstest
RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis

ENTRYPOINT ["/home/appaegis/bin/stresstest"]
