#!/usr/bin/env bash
# Renders the launchers of the plugin from the kit's templates into
# plugin/scripts/ of the consuming repository: launch-<SERVER> (Linux),
# launch-<SERVER>.cmd and launch.ps1 (Windows). The marketplace installs the
# plugin straight from the repository, so the rendered files are committed;
# `render.sh --check` (CI) fails when they are stale.
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

check=0
[[ ${1:-} != --check ]] || check=1

out=plugin/scripts
if ((check)); then
  out=$(mktemp -d)
  trap 'rm -rf "$out"' EXIT
fi
mkdir -p "$out"

render() { # render <template> <target>
  sed -e "s/@SERVER@/$SERVER/g" -e "s/@ORG@/$ORG/g" -e "s/@HOME_ENV@/$HOME_ENV/g" -e "s/@ARGS@/$ARGS/g" \
    "$KIT/templates/launcher/$1" >"$2"
}
render launch.sh.tmpl "$out/launch-$SERVER"
render launch.cmd.tmpl "$out/launch-$SERVER.cmd"
render launch.ps1.tmpl "$out/launch.ps1"
chmod 0755 "$out/launch-$SERVER"

if ((check)); then
  for f in "launch-$SERVER" "launch-$SERVER.cmd" launch.ps1; do
    cmp -s "$out/$f" "plugin/scripts/$f" ||
      { echo "render.sh: plugin/scripts/$f is stale, run the kit's scripts/render.sh and commit" >&2; exit 1; }
  done
  echo "launchers are up to date"
else
  echo "rendered plugin/scripts/launch-$SERVER, launch-$SERVER.cmd, launch.ps1"
fi
