#!/bin/bash

export GOPATH="`pwd`"
cd src/github.com/vbonnet/mobile-push-broadcaster/
go build main.go
./main