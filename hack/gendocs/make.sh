#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscodelabs/hugo-checker/hack/gendocs
go run main.go
popd
