#!/usr/bin/env bash
# Packs the plugin as it ships: text only, no binaries. The launcher in it
# downloads the binary pinned in release.json from GitHub Releases.
#   dist/<SERVER>-plugin.zip  - for Claude Desktop (Customize → Plugins → upload)
#   dist/marketplace/         - local marketplace for Claude Code:
#       claude plugin marketplace add ./dist/marketplace
#       claude plugin install <PLUGIN>@<PLUGIN>-local
# RELEASE_JSON=<file> packs that file as release.json instead of
# plugin/release.json; dev-install.sh uses it to pin a local build.
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

release=${RELEASE_JSON:-plugin/release.json}
version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$release" | head -n 1)
if [[ -z $version ]]; then
  echo "plugin.sh: no version in $release" >&2
  exit 1
fi
launcher=scripts/launch-$SERVER
[[ -f plugin/$launcher ]] || { echo "plugin.sh: plugin/$launcher is missing, run the kit's scripts/render.sh" >&2; exit 1; }

market=dist/marketplace
stage=$market/plugins/$PLUGIN
rm -rf "$market"
mkdir -p "$market/.claude-plugin" "$market/plugins"
cp -r plugin "$stage"
cp "$release" "$stage/release.json"
for f in LICENSE THIRD_PARTY_NOTICES.md; do
  [[ ! -f $f ]] || cp "$f" "$stage/"
done
chmod 0755 "$stage/$launcher"
# The manifest carries the pinned version, so hosts see an update.
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"${version#v}\"/" "$stage/.claude-plugin/plugin.json"

# grep -I skips binary files, so -L lists them (and empty files).
binaries=$(grep -rIL . "$stage" || true)
if [[ -n $binaries ]]; then
  echo "plugin.sh: the plugin must be text only, found:" >&2
  echo "$binaries" >&2
  exit 1
fi

cat >"$market/.claude-plugin/marketplace.json" <<JSON
{
  "name": "$PLUGIN-local",
  "description": "Local build of $SERVER for testing",
  "owner": { "name": "Anton Hokkanen" },
  "plugins": [{ "name": "$PLUGIN", "source": "./plugins/$PLUGIN" }]
}
JSON

# Entries without "./" (Desktop rejects them otherwise) and with the exec bit
# on the Linux launcher; mkzip needs no zip tool, so this works in Git Bash.
zip=dist/$SERVER-plugin.zip
rm -f "$zip"
(cd "$KIT" && go run ./cmd/mkzip -x "$launcher" "$ROOT/$zip" "$ROOT/$stage")
echo "$zip ($version)"
