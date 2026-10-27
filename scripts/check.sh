#!/usr/bin/env bash
# The repository checks of the reusable CI that are not Go tests:
#   - the launchers in plugin/scripts/ are what the kit renders;
#   - the Linux launcher and the scripts are executable in git (hosts copy
#     modes from git: without the bit Linux cannot start the server);
#   - no binary files in git (binaries come only from GitHub Releases);
#   - the plugin packs as text only.
# Run from CI after `go vet` and `go test`.
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

"$KIT/scripts/render.sh" --check

bad=$(git ls-files -s "plugin/scripts/launch-$SERVER" 'scripts/*.sh' |
  awk '$1 != "100755" { print $4 }')
[[ -z $bad ]] || { echo "::error::not 100755, run git update-index --chmod=+x on: $bad" >&2; exit 1; }

# grep -I skips binary files, so -L lists them, and empty files, which are fine.
bin=$(git ls-files -z | xargs -0 grep -IL -- . | while read -r f; do [ ! -s "$f" ] || echo "$f"; done || true)
[[ -z $bin ]] || { echo "::error::binary files in git:" >&2; echo "$bin" >&2; exit 1; }

"$KIT/scripts/plugin.sh"
