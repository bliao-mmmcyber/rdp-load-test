FROM golang:1.14 AS build-env

WORKDIR /var/appaegis/
ADD . .
RUN go mod tidy
RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./dist/guac cmd/guac/guac.go

#--------
FROM alpine:latest

ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-muslc-amd64 /bin/tini
RUN chmod +x /bin/tini
#---------
ENV ETCDCTL_ENDPOINT=http://127.0.0.1:2379
ENV ETCDCTL_USER=root:Appaegis1234

#-------------------
COPY --from=build-env /var/appaegis/dist/guac /home/appaegis/bin/guac

RUN mkdir -p /var/log/appaegis
WORKDIR /home/appaegis
#-------------------

LABEL name="guac" \
        version="1.0"   \
        description="guac"

ENTRYPOINT ["/bin/tini", "--"]
CMD ["/home/appaegis/bin/guac"]


