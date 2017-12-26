VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || cat $(CURDIR)/.version 2> /dev/null || echo v0)
BLDVER = module:$(MODULE),version:$(VERSION),build:$(shell date +"%Y%m%d.%H%M%S.%N.%z")
BASE = $(CURDIR)
MODULE = oath

.PHONY: all
all: version $(MODULE)

.PHONY: $(MODULE)
$(MODULE):| $(BASE)
	@go build -v -o $(BASE)/bin/$@

$(BASE):
	@mkdir -p $(dir $@)

# docker builds

.PHONY: locald oathd oathp __docker_oathd __docker_oathp
oathd: __docker_oathd prune
oathp: __checkenv __docker_oathp prune

# use kops id and secret
locald:
	docker build --rm -t $(MODULE) --build-arg awsrgn=ap-northeast-1 --build-arg awsid=$(OATH_ACCESS_KEY_ID) --build-arg awssec=$(OATH_SECRET_ACCESS_KEY) --build-arg version="$(BLDVER)" .

__docker_oathd:
	docker build -t $(IMAGE) --build-arg awsrgn=ap-northeast-1 --build-arg awsid=$(OATH_ACCESS_KEY_ID) --build-arg awssec=$(OATH_SECRET_ACCESS_KEY) --build-arg version="$(BLDVER)" .

__docker_oathp:
	@if test -z "$(PULLR_SNS_ARN)"; then echo "empty PULLR_SNS_ARN" && exit 1; fi; \
	if test -z "$(PULLR_SQS_URL)"; then echo "empty PULLR_SQS_URL" && exit 1; fi; \
	docker build -t $(PULLR_IMAGE_NAME) --build-arg awsrgn=ap-northeast-1 --build-arg awsid=$(AWS_ACCESS_KEY_ID) --build-arg awssec=$(AWS_SECRET_ACCESS_KEY) --build-arg pullrsns=$(PULLR_SNS_ARN) --build-arg pullrsqs=$(PULLR_SQS_URL) .;

__checkenv:
	if test -z "$(AWS_ACCESS_KEY_ID)"; then echo "empty AWS_ACCESS_KEY_ID" && exit 1; fi; \
	if test -z "$(AWS_SECRET_ACCESS_KEY)"; then echo "empty AWS_SECRET_ACCESS_KEY" && exit 1; fi

# docker run containers

.PHONY: on __on off __off
on: locald __on prune
off: __off prune

__on:
	@docker run --rm -d -p 8080:8080 --name $(MODULE) $(MODULE)

__off:
	@docker rm -f $(MODULE)

# misc

.PHONY: prune clean version list
prune:
	@docker system prune -f

clean:
	@rm -rfv bin; \
	docker rmi $(docker images --filter "dangling=true" -q --no-trunc); \
	exit 0

version:
	@echo "Version: $(VERSION)"

list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs
