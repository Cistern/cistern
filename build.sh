#!/bin/sh

set -e
cd ~/.go_workspace/src/github.com/Cistern/cistern
go build -o cistern-linux-amd64 ./cmd/cistern && mv cistern-linux-amd64 $CIRCLE_ARTIFACTS
GOOS=darwin GOARCH=amd64 go build -o cistern-darwin-amd64 ./cmd/cistern && mv cistern-darwin-amd64 $CIRCLE_ARTIFACTS
cd ui
npm i
npm run build
tar czvf cistern-ui-assets.tar.gz static && mv cistern-ui-assets.tar.gz $CIRCLE_ARTIFACTS
