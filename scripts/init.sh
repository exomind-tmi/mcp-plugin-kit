#!/usr/bin/env bash
# Scaffolds the plugin directory of a new server in the consuming repository:
# plugin/.claude-plugin/plugin.json, plugin/.mcp.json and a not-yet-released
# plugin/release.json, then renders the launchers. Files that exist are left
# alone. Needs mcp-plugin.env (see lib.sh) and PLUGIN_DESCRIPTION, e.g.
#   PLUGIN_DESCRIPTION='YouGile for Claude' scripts/init.sh
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

desc=${PLUGIN_DESCRIPTION:-$DESCRIPTION}
license=${LICENSE_ID:-UNLICENSED}
mkdir -p plugin/.claude-plugin

put() { # put <path> — writes stdin unless the file exists
  if [[ -e $1 ]]; then
    echo "kept $1"
    cat >/dev/null
  else
    cat >"$1"
    echo "wrote $1"
  fi
}

put plugin/.claude-plugin/plugin.json <<JSON
{
  "name": "$PLUGIN",
  "version": "0.0.0",
  "description": "$desc",
  "author": { "name": "Exomind Tmi", "url": "https://github.com/${REPO%%/*}" },
  "homepage": "https://github.com/$REPO",
  "repository": "https://github.com/$REPO",
  "license": "$license"
}
JSON

put plugin/.mcp.json <<JSON
{
  "mcpServers": {
    "$PLUGIN": {
      "command": "\${CLAUDE_PLUGIN_ROOT}/scripts/launch-$SERVER"
    }
  }
}
JSON

# Not released yet: the launcher says so instead of downloading anything.
put plugin/release.json <<'JSON'
{
  "version": "v0.0.0",
  "assets": {}
}
JSON

# Line endings: the Linux launcher must stay LF on every checkout; the Windows ones may be CRLF.
put .gitattributes <<GIT
* text=auto eol=lf
*.ps1 text eol=crlf
*.cmd text eol=crlf
plugin/scripts/launch-$SERVER text eol=lf
GIT

"$KIT/scripts/render.sh"
