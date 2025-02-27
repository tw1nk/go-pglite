package gopglite

import (
	"io"
	"os"

	"github.com/tetratelabs/wazero"
)

type config struct {
	// wazero runtime configuration
	stdout io.WriteCloser
	stderr io.WriteCloser
	tmpDir string
	devDir string

	// pglite configuration
	pguser     string
	pgdatabase string

	listen     bool
	listenPort int
}

func NewConfig() config {
	return config{
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		tmpDir:     "./tmp",
		devDir:     "./dev",
		pguser:     "postgres",
		pgdatabase: "postgres",
	}
}

func (c config) WithStdout(w io.WriteCloser) config {
	c.stdout = w
	return c
}

func (c config) WithStderr(w io.WriteCloser) config {
	c.stderr = w
	return c
}

func (c config) WithTmpDir(dir string) config {
	c.tmpDir = dir
	return c
}

func (c config) WithDevDir(dir string) config {
	c.devDir = dir
	return c
}

func (c config) WithUser(user string) config {
	c.pguser = user
	return c
}

func (c config) WithDatabase(database string) config {
	c.pgdatabase = database
	return c
}

// WithListen enables the server to listen on a port
// if port is 0, the server will not listen
func (c config) WithListen(port int) config {
	if port == 0 {
		return c
	}

	c.listen = true
	c.listenPort = port
	return c
}

func (c config) ModuleConfig() wazero.ModuleConfig {
	fsConfig := wazero.NewFSConfig().
		WithDirMount(c.tmpDir, "/tmp").
		WithDirMount(c.devDir, "/dev")

	return wazero.NewModuleConfig().
		WithStdout(c.stdout).
		WithStderr(c.stderr).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithFSConfig(fsConfig)
}
