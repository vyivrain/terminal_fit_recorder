#!/bin/bash

set -e

VERSION=${1:-"v1.0.0"}
APP_NAME="terminal_fit_recorder"
BUILD_DIR="build"

rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}/${APP_NAME} cmd/main.go
tar -czf ${BUILD_DIR}/${APP_NAME}-${VERSION}-linux-amd64.tar.gz -C ${BUILD_DIR} ${APP_NAME}
rm ${BUILD_DIR}/${APP_NAME}

GOOS=linux GOARCH=arm64 go build -o ${BUILD_DIR}/${APP_NAME} cmd/main.go
tar -czf ${BUILD_DIR}/${APP_NAME}-${VERSION}-linux-arm64.tar.gz -C ${BUILD_DIR} ${APP_NAME}
rm ${BUILD_DIR}/${APP_NAME}

GOOS=darwin GOARCH=amd64 go build -o ${BUILD_DIR}/${APP_NAME} cmd/main.go
tar -czf ${BUILD_DIR}/${APP_NAME}-${VERSION}-darwin-amd64.tar.gz -C ${BUILD_DIR} ${APP_NAME}
rm ${BUILD_DIR}/${APP_NAME}

GOOS=darwin GOARCH=arm64 go build -o ${BUILD_DIR}/${APP_NAME} cmd/main.go
tar -czf ${BUILD_DIR}/${APP_NAME}-${VERSION}-darwin-arm64.tar.gz -C ${BUILD_DIR} ${APP_NAME}
rm ${BUILD_DIR}/${APP_NAME}

GOOS=windows GOARCH=amd64 go build -o ${BUILD_DIR}/${APP_NAME}.exe cmd/main.go
cd ${BUILD_DIR} && zip ${APP_NAME}-${VERSION}-windows-amd64.zip ${APP_NAME}.exe && cd ..
rm ${BUILD_DIR}/${APP_NAME}.exe
