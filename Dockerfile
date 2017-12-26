FROM golang
ARG version
ADD . /go/src/github.com/mobingilabs/oath
WORKDIR /go/src/github.com/mobingilabs/oath
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags "-X github.com/mobingilabs/oath/cmd.version=$version" -a -installsuffix cgo -o oath .

FROM alpine:3.7
# Note that these env variables are visible via `docker history`, 'docker inspect`.
# Never upload to public registry; only ECR (current).
ARG awsrgn
ARG awsid
ARG awssec
RUN apk --no-cache add ca-certificates
ENV AWS_REGION=$awsrgn AWS_ACCESS_KEY_ID=$awsid AWS_SECRET_ACCESS_KEY=$awssec
WORKDIR /oath/
COPY --from=0 /go/src/github.com/mobingilabs/oath/oath .
ENTRYPOINT ["/oath/oath"]
CMD ["serve", "--logtostderr"]
