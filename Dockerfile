FROM golang:1.20

ADD src /src

WORKDIR /src

RUN go mod vendor
RUN go build

ENTRYPOINT ["/src/dockerfile-checker"]
