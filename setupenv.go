package gopglite

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed embedfiles/*
var embeddedFiles embed.FS

func setupEnv(cfg config) ([]byte, error) {
	// check if tar.gz already extracted; if not do so
	if _, err := os.Stat(filepath.Join(cfg.tmpDir, "pglite/base/PG_VERSION")); err != nil {
		fmt.Println("Extracting env....")

		tarGzFile, err := embeddedFiles.Open("embedfiles/pglite-wasi.tar.gz")
		if err != nil {
			return nil, err
		}
		defer tarGzFile.Close()

		gr, err := gzip.NewReader(tarGzFile)
		if err != nil {
			return nil, err
		}
		defer gr.Close()

		tr := tar.NewReader(gr)

		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			dest := filepath.Join("./", header.Name)

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(dest, os.FileMode(header.Mode)); err != nil {
					return nil, err
				}
			case tar.TypeReg:
				if err := os.MkdirAll(filepath.Dir(dest), os.FileMode(header.Mode)); err != nil {
					return nil, err
				}

				of, err := os.Create(dest)
				if err != nil {
					return nil, err
				}
				defer of.Close()

				if _, err := io.Copy(of, tr); err != nil {
					return nil, err
				}
			case tar.TypeSymlink:
				if err := os.Symlink(header.Linkname, dest); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown file type in tar: %c (%s)", header.Typeflag, header.Name)
			}
		}
	}

	// setup random
	if err := os.MkdirAll(cfg.devDir, 0755); err != nil {
		return nil, err
	}

	rf, err := os.Create(filepath.Join(cfg.devDir, "urandom"))
	if err != nil {
		return nil, err
	}
	defer rf.Close()

	rng := make([]byte, 128)
	if _, err := rand.Read(rng); err != nil {
		return nil, err
	}
	rf.Write(rng)

	// read in wasi blob
	return os.ReadFile(filepath.Join(cfg.tmpDir, "pglite/bin/postgres.wasi"))
}
