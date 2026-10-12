#!/bin/bash
set -e

VERSION="0.1.0-alpha"
OS="linux"
ARCH="amd64"
DIST_DIR="dist"
PACKAGE_NAME="bitrun-${VERSION}-${OS}-${ARCH}"

echo "Packaging BitRun ${VERSION} for ${OS}/${ARCH}..."

mkdir -p "${DIST_DIR}/${PACKAGE_NAME}/bin"

cp bin/bitrun "${DIST_DIR}/${PACKAGE_NAME}/bin/"
cp bin/llama-cli "${DIST_DIR}/${PACKAGE_NAME}/bin/"
cp bin/llama-server "${DIST_DIR}/${PACKAGE_NAME}/bin/"
cp bin/llama-quantize "${DIST_DIR}/${PACKAGE_NAME}/bin/"
cp README.md "${DIST_DIR}/${PACKAGE_NAME}/" 2>/dev/null || true
cp LICENSE "${DIST_DIR}/${PACKAGE_NAME}/" 2>/dev/null || true

cd "${DIST_DIR}"
tar -czvf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}"
cd ..

echo "Linux release archive created: ${DIST_DIR}/${PACKAGE_NAME}.tar.gz"
ls -lh "${DIST_DIR}/${PACKAGE_NAME}.tar.gz"
