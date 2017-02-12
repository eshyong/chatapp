#!/bin/bash

set -ex

pushd frontend
HTTPS=true npm start
