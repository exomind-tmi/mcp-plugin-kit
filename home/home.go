// Package home owns the state directory of an MCP server: its location, the
// files in it and how they are written. Nothing here depends on the current
// directory, which is system32 when Claude Desktop launches the server.
//
// The directory is ~/.mcp/<org>/<server>, or the absolute path in the
// environment variable <SERVER>_HOME, with the server name upper-cased and
// dashes turned into underscores (yougile-mcp -> YOUGILE_MCP_HOME). The
// launchers rendered by the kit use the same rule.
package home

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Home is the state directory.
type Home struct{ Dir string }

// envPrefix turns a server name into the prefix of its environment variables.
func envPrefix(server string) string {
	return strings.ToUpper(strings.ReplaceAll(server, "-", "_"))
}

// EnvHome is the variable that overrides the state directory of server.
func EnvHome(server string) string { return envPrefix(server) + "_HOME" }

// EnvLogLevel is the variable that selects the log level of server.
func EnvLogLevel(server string) string { return envPrefix(server) + "_LOG" }

// Resolve picks the state directory of server from <SERVER>_HOME or the user
// profile, without creating it.
func Resolve(org, server string) (Home, error) {
	env := EnvHome(server)
	if d := os.Getenv(env); d != "" {
		if !filepath.IsAbs(d) {
			return Home{}, fmt.Errorf("%s must be an absolute path, got %q", env, d)
		}
		return Home{filepath.Clean(d)}, nil
	}
	u, err := os.UserHomeDir()
	if err != nil {
		return Home{}, fmt.Errorf("locate user profile (or set %s): %w", env, err)
	}
	return Home{filepath.Join(u, ".mcp", org, server)}, nil
}

// Ensure creates the state directory and its fixed subdirectories (logs, bin),
// private to the user: keys and tokens live in them. On Unix a directory that
// exists already, made by an older version or by hand with a looser mode, is
// tightened too, which MkdirAll does not do. Windows has no use for it: the
// default home is in the user's profile, which only the user, the administrators
// and the system can read, and a home elsewhere has the access of the place.
func (h Home) Ensure() error {
	for _, d := range []string{h.Dir, h.LogsDir(), h.BinDir()} {
		if err := EnsureDir(d); err != nil {
			return err
		}
	}
	return nil
}

// EnsureDir creates dir (and parents) private to the user.
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return makePrivate(dir)
	}
	return nil
}

// makePrivate takes the access of the group and of others away from the directory
// d, if it has any. A directory that is the user's alone already is left as it is,
// so that one the user cannot chmod, such as a shared one that <SERVER>_HOME
// points to, does not stop every command; only a directory that stays open to
// others is an error.
func makePrivate(d string) error {
	st, err := os.Stat(d)
	if err != nil {
		return err
	}
	if st.Mode().Perm()&0o077 == 0 {
		return nil
	}
	if err := os.Chmod(d, 0o700); err != nil {
		return fmt.Errorf("make the state directory private: %w", err)
	}
	return nil
}

// BinDir is where the launcher keeps the downloaded binaries, one per version.
func (h Home) BinDir() string { return filepath.Join(h.Dir, "bin") }

// LogsDir is the directory of the log files, the launcher's among them.
func (h Home) LogsDir() string { return filepath.Join(h.Dir, "logs") }

// Log is the path of the log file name.
func (h Home) Log(name string) string { return filepath.Join(h.LogsDir(), name) }

// shareRetry covers a concurrent reader: on Windows os.ReadFile opens files
// without FILE_SHARE_DELETE, so a rename or delete during the read fails
// with a sharing violation.
const shareRetry = 250 * time.Millisecond

// Retry runs op until it succeeds or window has passed, returning the last
// error. It is for Windows file operations that fail while another process
// (a reader, an antivirus scanner, the indexer) briefly holds the file.
func Retry(window time.Duration, op func() error) error {
	deadline := time.Now().Add(window)
	for {
		err := op()
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// WriteAtomic replaces path with data via a temp file in the same directory.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op after a successful rename

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		return err
	}
	return Retry(shareRetry, func() error { return os.Rename(tmp, path) })
}
