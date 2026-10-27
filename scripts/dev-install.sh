#!/usr/bin/env bash
# Installs a dev build on this machine, for testing the plugin without a release:
#   1. builds the server for this OS (build.sh);
#   2. puts it into <home>/bin/<version>/, where the launcher of a plugin
#      pinned to this version finds it;
#   3. packs the plugin (plugin.sh) with a release.json that pins this
#      version: dist/<SERVER>-plugin.zip and dist/marketplace/.
# plugin/release.json stays untouched. <home> is $<SERVER>_HOME or
# ~/.mcp/<ORG>/<SERVER>, as for the launcher.
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

os=$(go env GOOS)
case $os in
  windows) exe=$SERVER.exe ;;
  linux) exe=$SERVER ;;
  *) echo "dev-install.sh: no launcher for $os" >&2; exit 1 ;;
esac
version=$("$KIT/scripts/build.sh" "$os")
built=dist/$os-amd64/$exe

# Like the launcher and the Go code: on Windows the home is USERPROFILE, which
# Git Bash's HOME need not be.
base=$HOME
[[ $os != windows || -z ${USERPROFILE:-} ]] || base=$(cygpath -u "$USERPROFILE")
home=${!HOME_ENV:-$base/.mcp/$ORG/$SERVER}
dst=$home/bin/$version
mkdir -p "$dst"
cp "$built" "$dst/.$exe.tmp"
mv -f "$dst/.$exe.tmp" "$dst/$exe"

# Named like a release asset and served from disk: if bin/ loses the build,
# the launcher downloads it again from here.
rel=dist/dev-release
rm -rf "$rel"
mkdir -p "$rel"
asset=$rel/${SERVER}_${version}_${os}_amd64${exe#"$SERVER"}
cp "$built" "$asset"
root=$(pwd -W 2>/dev/null || pwd) # Git Bash: C:/..., elsewhere /...
RELEASE_URL_BASE="file:///${root#/}/$rel" \
  "$KIT/scripts/release-json.sh" "$version" "$os-amd64=$asset" >dist/dev-release.json

RELEASE_JSON=dist/dev-release.json "$KIT/scripts/plugin.sh" >/dev/null
echo "installed $dst/$exe"
echo "packed dist/$SERVER-plugin.zip and dist/marketplace/ pinned to $version"
