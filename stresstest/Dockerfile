FROM golang:1.21 as build-env

ADD . /go/src/app
WORKDIR /go/src/app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN make go.build

#---------
FROM alpine:latest

ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-muslc-amd64 /bin/tini
RUN chmod +x /bin/tini

COPY --from=build-env /go/src/app/bin/stresstest /home/appaegis/bin/stresstest
RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis

ENTRYPOINT ["/bin/tini", "--"]
CMD ["/home/appaegis/bin/stresstest"]
