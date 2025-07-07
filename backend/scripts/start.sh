#!/bin/sh
#

set -ex

BIN_PATH=$(cd "$(dirname "$0")"; pwd -P)
WORK_PATH=${BIN_PATH}/../


cd ${WORK_PATH}

go build -o ./build/apps-backend server.go


./build/apps-backend $@

