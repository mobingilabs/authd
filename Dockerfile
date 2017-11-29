FROM golang

# Note that these env variables are visible via `docker history`, 'docker inspect`.
# Never upload to public registry; only ECR (current).
ARG awsrgn
ARG awsid
ARG awssec
ENV AWS_REGION=$awsrgn \
    AWS_ACCESS_KEY_ID=$awsid \
    AWS_SECRET_ACCESS_KEY=$awssec
ADD . /go/src/authd
WORKDIR /go/src/authd
RUN go build -v

ENTRYPOINT ["/go/src/authd/authd"]
CMD ["--logtostderr"]
