#!/usr/bin/env bash

ROOT=$(readlink -f "$(dirname "$0")/..")

target=$1
if [ -z "$target" ]; then
    echo "usage: $0 target"
    exit 1
fi

DOCKER_BUILDKIT=1 docker build -t "msgbounds:$target" --target="$target" "$ROOT"
