#!/usr/bin/env bash

set -u
set -o pipefail

image="${1}"

case "${image}" in
  "paketobuildpacks/build:full")
    echo "no preparation neeeded"
    ;;

  "ubuntu:22.04")
    apt-get update
    apt-get install -y jq ruby-full make wget
    ;;

  "macos")
    brew install coreutils
    ;;

  *)
    echo "no preparation neeeded"
    ;;
esac