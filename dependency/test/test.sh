#!/usr/bin/env bash

set -u
set -o pipefail

function hasParams() {
  local tarball_name version
  tarball_name="${1}"
  version="${2}"

  if [[ -z "${tarball_name}" ]]; then
    echo " ⛔ specify tarball_name as the first parameter"
    exit 1
  fi

  if [[ -z "${version}" ]]; then
    echo " ⛔ specify tarball_name as the first parameter"
    exit 1
  fi
}

function itExists() {
  echo -n "tarball exists"

  local tarball_name
  tarball_name="${1}"

  if [[ ! -f "${tarball_name}" ]]; then
    echo " ⛔ ${tarball_name} does not exist"
    exit 1
  fi

  echo "... ✅"
}

function itHasTheRightStructure() {
  local tarball_name version
  tarball_name="${1}"
  version="${2}"

  local source temp_file
  source="https://github.com/yarnpkg/yarn/releases/download/v${version}/yarn-v${version}.tar.gz"
  temp_file=$(mktemp)

  wget \
    --output-document "${temp_file}" \
    --quiet \
    "${source}"

  local source_dir sourceSHA256 actualSHA256
  source_dir=$(mktemp -d)

  tar --extract \
    --file="${temp_file}" \
    --directory="${source_dir}"

  local compiled_dir
  compiled_dir=$(mktemp -d)

  tar --extract \
    --file="${tarball_name}" \
    --directory="${compiled_dir}"

  pushd "${source_dir}/yarn-v${version}" > /dev/null
    sourceSHA256=$(find . -type f -exec sha256sum {} \; | sha256sum)
    sourceSHA256="${sourceSHA256::64}"
  popd > /dev/null

  pushd "${compiled_dir}" > /dev/null
    actualSHA256=$(find . -type f -exec sha256sum {} \; | sha256sum)
    actualSHA256="${actualSHA256::64}"
  popd > /dev/null

  if [[ "${sourceSHA256}" != "${actualSHA256}" ]]; then
    echo " ⛔ expected SHA256 of ${sourceSHA256} for ${source_dir}/yarn-v${version} but received ${actualSHA256} for ${compiled_dir}"
    exit 1
  fi

  echo "SHA256 matches... ✅"
}

function main(){
  hasParams "${@:-}"
  echo -n "1. " && itExists "${@:-}"
  echo -n "2. " && itHasTheRightStructure "${@:-}"
}

main "${@:-}"
