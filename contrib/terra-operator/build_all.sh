#!/bin/bash
set -euo pipefail

if [ "$#" -ne 1 ] || [ -z "$1" ] ; then
  echo "usage: ./build_all.sh <core-image-tag>" >&2
  exit 1
fi

VERSION="$1"

pushd ..

git checkout "$VERSION"
docker build -t "dochain/core:$VERSION" .
git checkout -

popd

docker build --build-arg "version=$VERSION" --build-arg chainid=cookie-1 -t "dochain/core-node:$VERSION" .
docker build --build-arg "version=$VERSION" --build-arg chainid=bombay-12 -t "dochain/core-node:$VERSION-testnet" .





