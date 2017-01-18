#!/bin/bash

set -ex

source ./env_vars
sudo -E bash -c "$(which chatapp)"
