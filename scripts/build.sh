#!/usr/bin/env bash
# Builds the server into dist/: dist/windows-amd64/<SERVER>.exe and
# dist/linux-amd64/<SERVER>, or only the OSes given (windows, linux).
#
# Version: $VERSION when set (CI passes the tag, e.g. v0.3.0); otherwise a dev
# build v<next patch after the latest v* tag>-dev.<unix time>, so every rebuild
# is semver-newer than the previous one and older than the next release.
# The binary gets it as main.version. Prints the version.
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

if [[ -z "${VERSION:-}" ]]; then
  last=$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || echo v0.0.0)
  IFS=. read -r major minor patch <<<"${last#v}"
  patch=${patch%%-*}
  VERSION="v${major}.${minor}.$((patch + 1))-dev.$(date +%s)"
fi
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "build.sh: VERSION '$VERSION' is not semver (vMAJOR.MINOR.PATCH[-pre])" >&2
  exit 1
fi

build() { # build <os> <arch>
  local ext=
  [[ $1 != windows ]] || ext=.exe
  CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" -o "dist/$1-$2/$SERVER$ext" "$MAIN_PKG"
}
want=("$@")
[[ ${#want[@]} -gt 0 ]] || want=(windows linux)
for os in "${want[@]}"; do
  case $os in windows | linux) ;; *) echo "build.sh: unknown OS '$os' (windows, linux)" >&2; exit 1 ;; esac
  for p in $PLATFORMS; do
    [[ ${p%%-*} != "$os" ]] || build "${p%%-*}" "${p#*-}"
  done
done
echo "$VERSION"
