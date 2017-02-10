#!/bin/bash

set -ex

source ./env_vars
pushd frontend
npm start
