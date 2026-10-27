package home

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnvNames(t *testing.T) {
	if got := EnvHome("yougile-mcp"); got != "YOUGILE_MCP_HOME" {
		t.Fatalf("EnvHome = %q", got)
	}
	if got := EnvLogLevel("whatsapp-mcp"); got != "WHATSAPP_MCP_LOG" {
		t.Fatalf("EnvLogLevel = %q", got)
	}
}

func TestResolveEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEMO_MCP_HOME", dir)
	h, err := Resolve("exomind-tmi", "demo-mcp")
	if err != nil || h.Dir != dir {
		t.Fatalf("Resolve = %v, %v; want %s", h, err, dir)
	}
}

func TestResolveRejectsRelativeOverride(t *testing.T) {
	t.Setenv("DEMO_MCP_HOME", "relative/dir")
	if _, err := Resolve("exomind-tmi", "demo-mcp"); err == nil || !strings.Contains(err.Error(), "DEMO_MCP_HOME") {
		t.Fatalf("err = %v, want a complaint naming DEMO_MCP_HOME", err)
	}
}

func TestResolveDefaultIsUnderUserProfile(t *testing.T) {
	t.Setenv("DEMO_MCP_HOME", "")
	profile := t.TempDir()
	t.Setenv("HOME", profile)
	t.Setenv("USERPROFILE", profile)
	h, err := Resolve("exomind-tmi", "demo-mcp")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(profile, ".mcp", "exomind-tmi", "demo-mcp"); h.Dir != want {
		t.Fatalf("Dir = %s, want %s", h.Dir, want)
	}
}

func TestEnsureCreatesSubdirsAndTightensLooseOnes(t *testing.T) {
	h := Home{Dir: filepath.Join(t.TempDir(), "state")}
	if err := os.MkdirAll(h.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.Ensure(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{h.Dir, h.LogsDir(), h.BinDir()} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			t.Fatalf("%s missing: %v", d, err)
		}
		if runtime.GOOS != "windows" && st.Mode().Perm() != 0o700 {
			t.Fatalf("%s mode = %o, want 700", d, st.Mode().Perm())
		}
	}
	if h.Log("a.log") != filepath.Join(h.Dir, "logs", "a.log") {
		t.Fatalf("Log = %s", h.Log("a.log"))
	}
}

func TestWriteAtomicReplacesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.json")
	for _, content := range []string{"one", "two"} {
		if err := WriteAtomic(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "two" {
		t.Fatalf("content = %q, %v", b, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestTailAndRotate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.log")
	if err := os.WriteFile(p, []byte("a\r\nb\r\nc\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Tail(p, 2)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("Tail = %q", got)
	}
	if err := os.WriteFile(p, make([]byte, rotateSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	RotateLog(p)
	if _, err := os.Stat(p + ".1"); err != nil {
		t.Fatalf("log was not rotated: %v", err)
	}
}

func TestOpenLogWritesAtConfiguredLevel(t *testing.T) {
	t.Setenv("DEMO_MCP_LOG", "warn")
	p := filepath.Join(t.TempDir(), "s.log")
	l, f, err := OpenLog("demo-mcp", p)
	if err != nil {
		t.Fatal(err)
	}
	l.Info("hidden")
	l.Warn("shown")
	f.Close()
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "hidden") || !strings.Contains(string(b), "shown") {
		t.Fatalf("log = %q", b)
	}
}
