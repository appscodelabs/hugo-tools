#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscodelabs/hugo-tools/hack/gendocs
go run main.go
popd
