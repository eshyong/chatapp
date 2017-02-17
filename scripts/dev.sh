#!/bin/bash

set -ex

source ./env_vars
make fetchdeps && make all && chatapp
