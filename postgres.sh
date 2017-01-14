#!/bin/bash

set -ex

# This script assumes we're using Homebrew's postgresql.
# Create the data directory
if [ ! -d data ]; then
    /usr/local/Cellar/postgresql/9.6.1/bin/initdb -D
fi

# Parse command line arguments
if [ "$1" = "start" ]; then
    brew services start postgresql
elif [ "$1" = "stop" ]; then
    brew services stop postgresql
elif [ "$1" = "restart" ]; then
    brew services restart postgresql
else
    echo Unknown argument "$1"
fi
