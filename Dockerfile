FROM build-base as base
FROM golang:1.15 as build-env

WORKDIR /go/src/app
COPY --from=base /go/src/golang-common /go/src/golang-common
ADD . .

RUN go mod tidy
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.commitID=$GIT_COMMIT" -o guac ./cmd/guac/

#---------
FROM alpine:latest

ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-muslc-amd64 /bin/tini
RUN chmod +x /bin/tini

COPY --from=build-env /go/src/app/guac /home/appaegis/bin/guac
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


