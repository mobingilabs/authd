FROM golang

ADD . /go/src/authd
WORKDIR /go/src/authd
RUN go build -v

ENTRYPOINT ["/go/src/authd/authd"]
