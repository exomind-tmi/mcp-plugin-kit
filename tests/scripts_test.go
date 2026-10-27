// Package tests drives the kit's scripts and launcher the way a consuming
// repository does: a temporary repository with an mcp-plugin.env, a fake
// release served from disk, and the rendered Linux launcher run against it.
package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const env = `SERVER=demo-mcp
PLUGIN=demo
ORG=exomind-tmi
REPO=exomind-tmi/demo-mcp
MAIN_PKG=./cmd/demo-mcp
`

func kitDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func needTools(t *testing.T, names ...string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the Linux launcher and the bash scripts are tested on Linux")
	}
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			t.Skipf("%s is not installed", n)
		}
	}
}

// consumer makes a repository root with mcp-plugin.env and renders the launchers.
func consumer(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "mcp-plugin.env"), env, 0o644)
	run(t, root, nil, "render.sh")
	return root
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func script(t *testing.T, name string) string { return filepath.Join(kitDir(t), "scripts", name) }

func run(t *testing.T, root string, extra []string, name string, args ...string) string {
	t.Helper()
	out, err := tryRun(t, root, extra, name, args...)
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return out
}

func tryRun(t *testing.T, root string, extra []string, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{script(t, name)}, args...)...)
	cmd.Env = append(os.Environ(), append([]string{"MCP_PLUGIN_ROOT=" + root}, extra...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestRenderSubstitutesNamesAndCheckDetectsStaleFiles(t *testing.T) {
	needTools(t, "bash", "sed")
	root := consumer(t)
	sh := read(t, filepath.Join(root, "plugin/scripts/launch-demo-mcp"))
	ps := read(t, filepath.Join(root, "plugin/scripts/launch.ps1"))
	for _, want := range []string{"DEMO_MCP_HOME", ".mcp/exomind-tmi/demo-mcp", `exec "$exe" stdio`} {
		if !strings.Contains(sh, want) {
			t.Errorf("sh launcher lacks %q", want)
		}
	}
	for _, want := range []string{"'demo-mcp.exe'", `.mcp\exomind-tmi\demo-mcp`, "$env:DEMO_MCP_HOME", "& $exe stdio"} {
		if !strings.Contains(ps, want) {
			t.Errorf("ps1 launcher lacks %q", want)
		}
	}
	for _, placeholder := range []string{"@SERVER@", "@ORG@", "@HOME_ENV@", "@ARGS@"} {
		if strings.Contains(sh+ps, placeholder) {
			t.Errorf("%s was left in a launcher", placeholder)
		}
	}
	for _, b := range []byte(ps) {
		if b > 127 {
			t.Fatal("launch.ps1 must stay ASCII: Windows PowerShell 5.1 reads BOM-less files in the ANSI code page")
		}
	}
	st, err := os.Stat(filepath.Join(root, "plugin/scripts/launch-demo-mcp"))
	if err != nil || st.Mode().Perm() != 0o755 {
		t.Fatalf("sh launcher mode = %v, %v; want 755", st, err)
	}

	run(t, root, nil, "render.sh", "--check")
	write(t, filepath.Join(root, "plugin/scripts/launch-demo-mcp"), "#!/bin/sh\n# edited by hand\n", 0o755)
	if out, err := tryRun(t, root, nil, "render.sh", "--check"); err == nil || !strings.Contains(out, "stale") {
		t.Fatalf("render --check passed on a stale launcher: %v\n%s", err, out)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestReleaseJSONShapeAndErrors(t *testing.T) {
	needTools(t, "bash", "sha256sum")
	root := consumer(t)
	bin := filepath.Join(root, "demo-mcp_v1.2.3_linux_amd64")
	write(t, bin, "payload", 0o755)
	sum := sha256.Sum256([]byte("payload"))

	out := run(t, root, nil, "release-json.sh", "v1.2.3", "linux-amd64="+bin)
	for _, want := range []string{
		`"version": "v1.2.3"`,
		`"url": "https://github.com/exomind-tmi/demo-mcp/releases/download/v1.2.3/demo-mcp_v1.2.3_linux_amd64"`,
		`"sha256": "` + hex.EncodeToString(sum[:]) + `"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("release.json lacks %s:\n%s", want, out)
		}
	}
	if _, err := tryRun(t, root, nil, "release-json.sh", "1.2.3", "linux-amd64="+bin); err == nil {
		t.Error("a version without the v prefix was accepted")
	}
	if _, err := tryRun(t, root, nil, "release-json.sh", "v1.2.3", "linux-amd64="+filepath.Join(root, "missing")); err == nil {
		t.Error("a missing asset file was accepted")
	}
}

// launch runs the rendered launcher against a release.json and returns its
// stdout, stderr and error.
func launch(t *testing.T, root, state, releaseJSON string) (stdout, stderr string, err error) {
	t.Helper()
	write(t, filepath.Join(root, "plugin/release.json"), releaseJSON, 0o644)
	cmd := exec.Command(filepath.Join(root, "plugin/scripts/launch-demo-mcp"))
	cmd.Env = append(os.Environ(), "DEMO_MCP_HOME="+state)
	var so, se strings.Builder
	cmd.Stdout, cmd.Stderr = &so, &se
	err = cmd.Run()
	return so.String(), se.String(), err
}

func TestLauncherDownloadsVerifiesAndRunsTheBinary(t *testing.T) {
	needTools(t, "bash", "sed", "curl", "flock", "sha256sum")
	if runtime.GOARCH != "amd64" {
		t.Skip("the launcher knows only linux-amd64 on Linux")
	}
	root := consumer(t)
	state := t.TempDir()
	payload := "#!/bin/sh\necho \"args:$*\"\n"
	asset := filepath.Join(root, "dist", "demo-mcp_v1.2.3_linux_amd64")
	write(t, asset, payload, 0o644)
	sum := sha256.Sum256([]byte(payload))
	good := `{"version":"v1.2.3","assets":{"linux-amd64":{"url":"file://` + asset + `","sha256":"` + hex.EncodeToString(sum[:]) + `"}}}`

	// A release that has no build yet says so and starts nothing.
	if out, errOut, err := launch(t, root, state, `{"version":"v0.0.0","assets":{}}`); err == nil || out != "" || !strings.Contains(errOut, "no linux-amd64 build") {
		t.Fatalf("unreleased: err=%v stdout=%q stderr=%q", err, out, errOut)
	}

	// A wrong hash installs nothing: the target never holds an unverified file.
	bad := strings.Replace(good, hex.EncodeToString(sum[:])[:8], "00000000", 1)
	if out, errOut, err := launch(t, root, state, bad); err == nil || out != "" || !strings.Contains(errOut, "SHA-256 mismatch") {
		t.Fatalf("mismatch: err=%v stdout=%q stderr=%q", err, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(state, "bin", "v1.2.3", "demo-mcp")); err == nil {
		t.Fatal("a binary with a wrong hash was installed")
	}

	// The good release is downloaded, installed and started with the configured
	// arguments; stdout carries only the binary's output.
	out, errOut, err := launch(t, root, state, good)
	if err != nil || out != "args:stdio\n" {
		t.Fatalf("launch: err=%v stdout=%q stderr=%q", err, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(state, "bin", "v1.2.3", "demo-mcp")); err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	// The second start needs no download: the file may be gone.
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if out, _, err := launch(t, root, state, good); err != nil || out != "args:stdio\n" {
		t.Fatalf("second launch: err=%v stdout=%q", err, out)
	}
}

func TestPluginPacksTextOnlyWithTheRenderedLauncher(t *testing.T) {
	needTools(t, "bash", "sed", "go")
	root := consumer(t)
	write(t, filepath.Join(root, "plugin/.claude-plugin/plugin.json"), `{"name":"demo","version": "0.0.0"}`, 0o644)
	write(t, filepath.Join(root, "plugin/.mcp.json"), `{}`, 0o644)
	write(t, filepath.Join(root, "plugin/release.json"), `{"version":"v1.2.3","assets":{}}`, 0o644)
	out := run(t, root, nil, "plugin.sh")
	if !strings.Contains(out, "dist/demo-mcp-plugin.zip (v1.2.3)") {
		t.Fatalf("plugin.sh output: %s", out)
	}
	m := read(t, filepath.Join(root, "dist/marketplace/plugins/demo/.claude-plugin/plugin.json"))
	if !strings.Contains(m, `"version": "1.2.3"`) {
		t.Fatalf("manifest version not pinned: %s", m)
	}
	if _, err := os.Stat(filepath.Join(root, "dist/demo-mcp-plugin.zip")); err != nil {
		t.Fatal(err)
	}

	// A binary in the plugin is refused.
	write(t, filepath.Join(root, "plugin/blob"), "\x00\x01\x02", 0o644)
	if out, err := tryRun(t, root, nil, "plugin.sh"); err == nil || !strings.Contains(out, "text only") {
		t.Fatalf("a binary file in the plugin was accepted: %v\n%s", err, out)
	}
}
