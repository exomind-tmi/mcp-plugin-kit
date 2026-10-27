# mcp-plugin-kit

Shared tooling to build, release and install local MCP servers written in Go as Claude plugins.

The user gets a **plugin** (`claude plugin install <name>@exomind`), not a binary. The plugin is text only:
a manifest, `.mcp.json`, optional skills and a launcher. On first start the launcher downloads the server
binary pinned in `plugin/release.json` from GitHub Releases, checks its SHA-256 and runs it. No interpreter,
no runtime on the user's machine.

This kit holds what every such server repeats, so it is written once:

| Part                    | What it is                                                                              |
|-------------------------|-----------------------------------------------------------------------------------------|
| `home/`, `proc/`        | Go packages: the state directory `~/.mcp/<org>/<server>` with logs and private modes; an exclusive file lock and detached spawn |
| `templates/launcher/`   | Launchers for Linux (`sh`) and Windows (`.cmd` + `launch.ps1`), rendered per server     |
| `scripts/`              | `init`, `render`, `build`, `release-json`, `plugin`, `dev-install`, `notices`, `check`  |
| `.github/workflows/`    | Reusable workflows: `go-ci.yml` and `go-release.yml`                                    |
| `cmd/mkzip`             | Zips the plugin the way Claude Desktop accepts it (no `./` prefix, exec bit kept)       |

It is not an MCP SDK (servers use [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk))
and not a server generator.

## Using it

A server repository has a `mcp-plugin.env` at its root:

```sh
SERVER=yougile-mcp                 # binary, state directory and release asset prefix
PLUGIN=yougile                     # plugin name in the marketplace
ORG=exomind-tmi                    # state directory: ~/.mcp/<ORG>/<SERVER>
REPO=exomind-tmi/yougile-mcp       # GitHub repository that publishes the releases
MAIN_PKG=./cmd/yougile-mcp         # Go package of the binary
ARGS=stdio                         # what the launcher starts the binary with (default stdio)
PLATFORMS="windows-amd64 linux-amd64"   # default
LICENSE_ID=0BSD                    # SPDX id, for the notices (optional)
```

1. `scripts/init.sh` writes `plugin/.claude-plugin/plugin.json`, `plugin/.mcp.json` and a not-yet-released
   `plugin/release.json`, and renders the launchers. Files that exist are kept.
2. The marketplace installs the plugin straight from the repository, so the rendered launchers in
   `plugin/scripts/` are committed. `scripts/render.sh` renders them again; CI fails when they are stale.
3. The server imports `home` for its state directory (`home.Resolve("exomind-tmi", "yougile-mcp")`
   honours `YOUGILE_MCP_HOME`, the same rule the launcher applies) and sets `main.version`, which `build.sh`
   fills with `-ldflags -X main.version=…`.
4. CI and release call the reusable workflows:

```yaml
# .github/workflows/ci.yml
jobs:
  ci:
    uses: exomind-tmi/mcp-plugin-kit/.github/workflows/go-ci.yml@main
    permissions:
      contents: read
```

```yaml
# .github/workflows/release.yml   (on: push: tags: ['v*'])
jobs:
  release:
    uses: exomind-tmi/mcp-plugin-kit/.github/workflows/go-release.yml@main
    permissions:
      contents: write
```

Pin `@main` and `kit-ref` to a tag once the kit has releases.

Scripts run from anywhere in the repository (`MCP_PLUGIN_ROOT` overrides the root). Locally, call them from
a checkout of the kit: `../mcp-plugin-kit/scripts/dev-install.sh` builds the server, puts it where the
launcher looks and packs a local marketplace (`dist/marketplace/`) and a zip for Claude Desktop.

## The release flow

A `v*` tag on `main` builds the binaries, publishes a GitHub Release (`<SERVER>_<tag>_<os>_<arch>[.exe]`,
`SHA256SUMS`) and then adds a follow-up commit on `main` that pins `plugin/release.json` and the version in
`plugin.json` to that release. Hosts install the plugin from `main`, so they point at a release only once its
binaries are published. See the header of `go-release.yml` for the details and the two optional hooks
(`scripts/release-extra.sh`, `scripts/pin-check.sh`).

`release.json` is parsed by the Linux launcher with `sed`: keep to the shape `release-json.sh` prints.

## Development

```
go vet ./... && go test ./...
```

`tests/` renders launchers for a demo server and runs them against a fake release on disk. The workflows
and the Windows launcher are checked only by GitHub Actions (`ci.yml`).

## License

[0BSD](LICENSE) © 2026 Exomind Tmi
