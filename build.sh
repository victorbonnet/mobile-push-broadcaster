#!/bin/bash

export GOPATH="`pwd`"
rm -Rf bin/*
echo "install web_logs"
go install github.com/vbonnet/mobile-push-broadcaster/web_logs

if [ "$1" = "skipTests" ]; then
   echo "skip test dao"
else
   echo "test dao"
   go test github.com/vbonnet/mobile-push-broadcaster/dao
fi

echo "install dao"
go install github.com/vbonnet/mobile-push-broadcaster/dao
echo "install mobile-push-broadcaster"
go install github.com/vbonnet/mobile-push-broadcaster

echo "copy static files"
cp -r src/github.com/vbonnet/mobile-push-broadcaster/config.json ./bin/
cp -r src/github.com/vbonnet/mobile-push-broadcaster/web ./bin/
