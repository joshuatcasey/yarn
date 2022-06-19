#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROGDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly BUILDPACKDIR="$(cd "${PROGDIR}/.." && pwd)"


function main() {
  pushd "${BUILDPACKDIR}/dependency"
    versions=$(make retrieve)

    for v in $(echo ${versions} | jq -rc '.[]'); 
      do 
        mkdir -p $PWD/metadata-${v}.json 
        mkdir -p $PWD/bundler-${v}.tgz 
        make metadata version=${v} output=$PWD/metadata-${v}.json/metadata-${v}.json 
        make compile version=${v} tarball_name=bundler-${v}.tgz/bundler-${v}.tgz os=macos-latest 
        make test version=${v} tarball_name=bundler-${v}.tgz/bundler-${v}.tgz
        # TODO: add an upload step
        # TODO: update metadata.json
      done
  popd

  pushd "${BUILDPACKDIR}/libdependency"
    make assemble id=bundler artifactPath=$PWD/../dependency \
    buildpackTomlPath=$PWD/../buildpack.toml
  popd

  git status
}

main "${@:-}"
