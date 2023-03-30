![cod cov](https://appaegis-codecoverage.s3.amazonaws.com/guac/badge.svg)

# guac

A port of the [Apache Guacamole client](https://github.com/apache/guacamole-client) to Go.

Apache Guacamole provides access to your desktop using remote desktop protocols in your web browser without any plugins.

[![GoDoc](https://godoc.org/github.com/wwt/guac?status.svg)](http://godoc.org/github.com/wwt/guac)
[![Go Report Card](https://goreportcard.com/badge/github.com/wwt/guac)](https://goreportcard.com/report/github.com/wwt/guac)
[![Build Status](https://travis-ci.org/wwt/guac.svg?branch=master)](https://travis-ci.org/wwt/guac)

## Development

First start guacd in a container, for example:

```sh
docker run --name guacd -d -p 4822:4822 guacamole/guacd
```

Next run the example main:

```sh
go run cmd/guac/guac.go
```

Now you can connect with [the example Vue app](https://github.com/wwt/guac-vue)

## Acknowledgements

Initially forked from https://github.com/johnzhd/guacamole_client_go which is a direct rewrite of the Java Guacamole
client. This project no longer resembles that one but it helped it get off the ground!

Some of the comments are taken directly from the official Apache Guacamole Java client.


## Deploy in Cloud

- Step 1 local build image by docker

Makefile will create three images name `980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac:<BRANCH_NAME>, 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac/transcode:<BRANCH_NAME>,980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac/stresstest:<BRANCH_NAME>,`. You can check if the image is on Elastic Container Register 

```
make clean build
make clean release
```

- guac: https://us-east-1.console.aws.amazon.com/ecr/repositories/private/980993447824/appaegis/guac?region=us-east-1
- transcode: https://us-east-1.console.aws.amazon.com/ecr/repositories/private/980993447824/appaegis/guac/transcode?region=us-east-1





- Step 2 Connect to `dev dp`

This machine is build on ec2 `dev dp`, there are few ways to connect to the instance.
1. Login the machine with a `private_key.pem` and username `ubuntu`. Please ask your co-worker for `private_key.pem` of `dep dp` if you don't have one.

```
ssh -i private_key.pem ubuntu@52.205.107.255
```
2. Login to our corp idp (https://corp.appaegis.net/ssh_connection), click `Applauncher/SSH`, select `dp dev ssh`, then click `Connect`.
3. Open Mammoth Browser, start `SSH/unity-dev dp`, select connect. After that, open a new termial, paste
`ssh ssh unity-dev-dp` to connect to the machine.

This server docker file is in `deploy/newdepoly`.
```
cd deploy/newdeploy
```

- Step 3 Update server image

Rewirte the `docker-compose.yaml` according to your service. In the repo, change `guac/transcoding` image url.

```
guac:
  depends_on:
    ...
  image: "980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac:<BRANCH_NAME>"

transcoding:
  depends_on:
    ...
  image: "980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac/transcoding:<BRANCH_NAME>"

```

- Step 4 relaunch docker image

Restart docker image to update environment

```
docker-compose stop guac
docker-compose up -d guac

docker-compose stop transcoding
docker-compose up -d transcoding

```

If you want to see the logs of the server
```
docker-compose logs -f guac/transcoding
```

- Step 5 update your image (Optional)

If you want to update your image, pull the image and restart `guac/transcoding`


```
dokcer pull 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac:<BRANCH_NAME>
docker-compose stop guac
docker-compose up -d guac

dokcer pull 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac/transcoding:<BRANCH_NAME>
docker-compose stop transcoding
docker-compose up -d transcoding

```
