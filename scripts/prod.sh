#!/bin/bash

set -ex

# Source environment variables
source ./env_vars

# Build frontend code
pushd frontend
npm run build
popd

# Make and run go server
make fetchdeps && make
sudo -E bash -c "$(which chatapp)"
