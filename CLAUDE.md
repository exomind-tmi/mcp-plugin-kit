# mcp-plugin-kit

## Commits

- One line, Conventional Commits 1.0.0: `<type>(<optional scope>): <imperative, lowercase description>`.
  A `Co-Authored-By` trailer follows after a blank line (decision of the owner for this repository).

## Rules

- The launchers are the product: `templates/launcher/*` become files users run on their machines. A change
  there changes every plugin built on the kit, so keep `launch.ps1.tmpl` ASCII (Windows PowerShell 5.1 reads
  BOM-less files in the ANSI code page) and keep stdout clean (it is the MCP channel).
- Scripts are bash and run on Linux, in Git Bash on Windows and in CI. `shellcheck` clean, mode 100755.
- Nothing here is specific to one server. A name, an organization or a path comes from `mcp-plugin.env`.
- Go: standard library plus `golang.org/x/sys`. Every other dependency needs a reason.
- `go vet ./... && go test ./...` before committing; tests are Linux-only where they run bash.
