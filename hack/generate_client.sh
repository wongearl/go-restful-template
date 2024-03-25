#!/bin/bash

set -e

GV="$1"
# github.com/wongearl/go-restful-template (到此处止，前面必须是go mod名，不能是项目目录名，否则无法生成lister,informers)  /pkg/api 
./hack/generate_group.sh "client,lister,informer" github.com/wongearl/go-restful-template/pkg/client/ai github.com/wongearl/go-restful-template/pkg/api "${GV}" --output-base=./  -h "$PWD/hack/boilerplate.go.txt"
rm -rf pkg/client/ai
mv github.com/wongearl/go-restful-template/pkg/client/ai pkg/client
rm -rf github.com