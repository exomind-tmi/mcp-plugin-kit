#!/usr/bin/env bash
# Sourced by the kit's scripts: finds the consuming repository, reads its
# mcp-plugin.env and derives the names the scripts share. Not run on its own.
#
# The repository is $MCP_PLUGIN_ROOT, or the git root of the current directory.
# mcp-plugin.env there holds shell variable assignments:
#
#   SERVER     binary, state directory and release asset prefix  (yougile-mcp)
#   PLUGIN     plugin name in the marketplace                    (yougile)
#   ORG        organization segment of ~/.mcp/<ORG>/<SERVER>     (exomind-tmi)
#   REPO       GitHub repository that publishes the releases     (exomind-tmi/yougile-mcp)
#   MAIN_PKG   Go package of the binary                          (./cmd/yougile-mcp)
#   ARGS       arguments the launcher starts the binary with     (stdio)
#   PLATFORMS  <os>-<arch> builds; windows-amd64 and linux-amd64 only for now
#   DESCRIPTION  one line for the local test marketplace
#   LICENSE_ID   SPDX id of the repository's license, for the notices (optional)
set -euo pipefail

KIT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
ROOT=${MCP_PLUGIN_ROOT:-$(git rev-parse --show-toplevel)}
cd "$ROOT"
command -v go >/dev/null || PATH="/c/Program Files/Go/bin:$PATH" # Git Bash on Windows

[[ -f mcp-plugin.env ]] || { echo "$(basename "$0"): no mcp-plugin.env in $ROOT" >&2; exit 1; }
# shellcheck source=/dev/null
source mcp-plugin.env

for v in SERVER PLUGIN ORG REPO MAIN_PKG; do
  [[ -n ${!v:-} ]] || { echo "mcp-plugin.env: $v is not set" >&2; exit 1; }
done
ARGS=${ARGS:-stdio}
PLATFORMS=${PLATFORMS:-windows-amd64 linux-amd64}
DESCRIPTION=${DESCRIPTION:-$PLUGIN for Claude}
LICENSE_ID=${LICENSE_ID:-}
[[ $SERVER =~ ^[a-z][a-z0-9-]*$ && $PLUGIN =~ ^[a-z][a-z0-9-]*$ && $ORG =~ ^[a-z0-9][a-z0-9-]*$ ]] ||
  { echo "mcp-plugin.env: SERVER, PLUGIN and ORG must be lower-case words with dashes" >&2; exit 1; }
for p in $PLATFORMS; do
  case $p in windows-amd64 | linux-amd64) ;; *) echo "mcp-plugin.env: platform '$p' has no launcher" >&2; exit 1 ;; esac
done

# HOME_ENV is the variable that overrides the state directory: the same rule
# as home.EnvHome in the Go module.
HOME_ENV=$(printf '%s' "$SERVER" | tr 'a-z-' 'A-Z_')_HOME
