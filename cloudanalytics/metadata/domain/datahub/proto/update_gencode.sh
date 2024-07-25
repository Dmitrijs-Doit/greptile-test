#!/bin/sh

cp ../../../../../../../../infra/services/datahub-api/proto/*.proto .
protoc --go_out=paths=source_relative:. event.proto
rm *.proto
