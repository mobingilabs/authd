FROM golang

# Note that these env variables are visible via `docker history`, 'docker inspect`.
# Never upload to public registry; only ECR (current).
ARG version
ARG awsrgn
ARG awsid
ARG awssec
ENV AWS_REGION=$awsrgn \
    AWS_ACCESS_KEY_ID=$awsid \
    AWS_SECRET_ACCESS_KEY=$awssec
ADD . /go/src/github.com/mobingilabs/authd
WORKDIR /go/src/github.com/mobingilabs/authd
RUN go build -v -ldflags "-X github.com/mobingilabs/authd/cmd.version=$version"

ENTRYPOINT ["/go/src/github.com/mobingilabs/authd/authd"]
CMD ["serve", "--logtostderr"]
