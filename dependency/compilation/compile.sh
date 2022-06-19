#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROGDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly NAME="bundler"

function main() {
  local version output_dir target exploded_directory tarball_name
  version="${1}"
  output_dir="${2}"
  target="${3}"

  echo "version=${version}"
  echo "output_dir=${output_dir}"
  echo "target=${target}"

  exploded_directory=$(mktemp -d)

  local source temp_file
  source="https://github.com/yarnpkg/yarn/releases/download/v${version}/yarn-v${version}.tar.gz"
  temp_file=$(mktemp)

  wget \
    --output-document "${temp_file}" \
    --quiet \
    "${source}"

  tar --extract \
    --verbose \
    --file="${temp_file}" \
    --directory="${exploded_directory}" \
    --strip-components=1

  pushd "${exploded_directory}" > /dev/null
    tar --create \
      --verbose \
      --file="${output_dir}/TEMP.tgz" \
      --gzip \
      .

# These tar options are not available on OSX
#      --owner=0 \
#      --group=0 \

  popd > /dev/null

  pushd "${output_dir}" > /dev/null
    sha256=$(sha256sum "TEMP.tgz")
    sha256="${sha256::8}"

    tarball_name="yarn_${version}_linux_noarch_${target}_${sha256}.tgz"
    mv "TEMP.tgz" "${tarball_name}"

    sha256sum "${tarball_name}" > "${tarball_name}.sha256"
  popd > /dev/null

  rm -rf "${exploded_directory}"
}

main "${@:-}"
