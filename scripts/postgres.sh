#!/bin/bash

set -ex

# This script assumes we're using Homebrew's postgresql.
# Create the data directory
if [ ! -d data ]; then
    /usr/local/Cellar/postgresql/9.6.1/bin/initdb --pgdata=$(pwd)/data
fi

# Parse command line arguments
if [ "$1" = "start" ] || [ "$1" = "stop" ] || [ "$1" = "restart" ]; then
    brew services "$1" postgresql
elif [ -z "$1" ]; then
    echo "Usage: ./postgres.sh start|stop|restart"
else
    echo Unknown argument "$1"
fi
