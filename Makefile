SHELL:=/bin/bash

.PHONY: vet clean test

MKFILE_PATH := $(abspath $(lastword ${MAKEFILE_LIST}))
MKFILE_DIR := $(shell dirname ${MKFILE_PATH})
ALL_FILES := $(shell find ${MKFILE_DIR}/{cmd,pkg} -type f -name "*.go")

TARGET = ${MKFILE_DIR}/bin/xdplbmgmt

build: vet ${TARGET}
${TARGET}: ${ALL_FILES}
	@cd ${MKFILE_DIR} && \
		CGO_ENABLED=0 go build -v \
  	-o ${TARGET} ${MKFILE_DIR}/cmd/frontend.go


vet:
	@cd ${MKFILE_DIR} && go vet ./{cmd,pkg}/...

clean:
	@rm -f ${TARGET}
