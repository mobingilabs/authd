VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || cat $(CURDIR)/.version 2> /dev/null || echo v0)
BASE = $(CURDIR)

.PHONY: all
all: version authd

.PHONY: authd
authd:| $(BASE)
	@go build -v -o $(BASE)/bin/$@

$(BASE):
	@mkdir -p $(dir $@)

# docker builds

.PHONY: locald authdd authdp __docker_authdd __docker_authdp
authdd: __checkenv __docker_authdd prune
authdp: __checkenv __docker_authdp prune

# use kops id and secret
locald:
	@docker build --rm -t authd:dev --build-arg awsrgn=ap-northeast-1 --build-arg awsid=$(KOPS_ACCESS_KEY_ID) --build-arg awssec=$(KOPS_SECRET_ACCESS_KEY) .; \
	make prune;

__docker_authdd:
	@if test -z "$(DEV_PULLR_SNS_ARN)"; then echo "empty DEV_PULLR_SNS_ARN" && exit 1; fi; \
	if test -z "$(DEV_PULLR_SQS_URL)"; then echo "empty DEV_PULLR_SQS_URL" && exit 1; fi; \
	docker build -t $(PULLR_IMAGE_NAME) --build-arg awsrgn=ap-northeast-1 --build-arg awsid=$(AWS_ACCESS_KEY_ID) --build-arg awssec=$(AWS_SECRET_ACCESS_KEY) --build-arg pullrsns=$(DEV_PULLR_SNS_ARN) --build-arg pullrsqs=$(DEV_PULLR_SQS_URL) .;

__docker_authdp:
	@if test -z "$(PULLR_SNS_ARN)"; then echo "empty PULLR_SNS_ARN" && exit 1; fi; \
	if test -z "$(PULLR_SQS_URL)"; then echo "empty PULLR_SQS_URL" && exit 1; fi; \
	docker build -t $(PULLR_IMAGE_NAME) --build-arg awsrgn=ap-northeast-1 --build-arg awsid=$(AWS_ACCESS_KEY_ID) --build-arg awssec=$(AWS_SECRET_ACCESS_KEY) --build-arg pullrsns=$(PULLR_SNS_ARN) --build-arg pullrsqs=$(PULLR_SQS_URL) .;

__checkenv:
	if test -z "$(AWS_ACCESS_KEY_ID)"; then echo "empty AWS_ACCESS_KEY_ID" && exit 1; fi; \
	if test -z "$(AWS_SECRET_ACCESS_KEY)"; then echo "empty AWS_SECRET_ACCESS_KEY" && exit 1; fi

# docker run containers

.PHONY: on __on off __off
on: __on prune
off: __off prune

__on:
	@docker-compose up -d --build

__off:
	@docker-compose down

# misc

.PHONY: prune clean version list
prune:
	@docker system prune -f; \
	docker rmi $(docker images --filter "dangling=true" -q --no-trunc); \
	exit 0

clean:
	@rm -rfv bin

version:
	@echo "Version: $(VERSION)"

# From https://stackoverflow.com/questions/4219255/how-do-you-get-the-list-of-targets-in-a-makefile
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs