#!/usr/bin/make
#
# Make all the things
#
#

GO      = go
GODOC   = godoc
GOFMT   = gofmt
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")
NOW=$(shell date +%Y%m%dT%H%M%S)

all: xformer

xformer:  transformer/*.go go.mod Makefile
	CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-N -l" -a -installsuffix cgo -o xformer ./transformer

# docker: xformer
# 	cd docker && make
# 	touch docker


.PHONY: clean
clean:
	@echo "$(M) Cleaning generated files"
	$(Q)rm -f xformer

release: production

.PHONY: run
run: xformer
	mv schedule.json sched-old.json
	time runxformer.sh >schedule.json

.PHONY: dump
dump: xformer
	mv guidebook.json guide-old.json
	time runxformer.sh -dump >guidebook.json


# .PHONY: production
production: staging
ifneq ($(shell grep '^replace ' go.mod | wc -l),0)
	@echo "====> The code references local replacement libraries: refusing to release to production"
	@/bin/false
endif
ifneq ($(shell git status | grep Untracked | wc -l),0)
	@git status
	@echo "====> There are untracked local files: refusing to release to production"
	@/bin/false
endif
ifneq ($(shell git diff-index --name-only HEAD | wc -l),0)
	@git status
	@echo "====> There are uncommitted local changes: refusing to release to production"
	@/bin/false
endif
ifneq ($(shell git status | grep 'branch is up to date' | wc -l),1)
	@git status
	@echo "====> The local branch is not up to date: refusing to release to production"
	@/bin/false
endif
	# These both need to be forced, since we explicitly move this tag
	git tag -f production
	git push -f `git remote` production
	git tag prod-$(NOW) && git push `git remote` prod-$(NOW)
	touch production

# .PHONY: staging
staging: xformer
ifneq ($(shell grep '^replace ' go.mod | wc -l),0)
	@echo "====> The code references local replacement libraries: refusing to release to staging"
	@/bin/false
endif
	touch staging

# Dependency management
.PHONY: modules
modules:
	GOPRIVATE=gitlab.com/karora/* go get -u `grep '^[[:space:]]' go.mod | grep -v indirect | cut -f1 -d' '`
	go mod tidy

