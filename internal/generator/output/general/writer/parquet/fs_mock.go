package parquet

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/arrow-go/v18/parquet"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func newFileSystemMock() *fsMock {
	return &fsMock{
		m: afero.NewMemMapFs(),
	}
}

type fsMock struct {
	m afero.Fs
}

func (f *fsMock) NewFileWriter(fileName string) (io.WriteCloser, error) {
	fw, err := f.m.Create(fileName)
	if err != nil {
		return nil, errors.Errorf("failed to create file in memory fs mock: %v", err)
	}

	return fw, nil
}

func (f *fsMock) NewLocalFileReader(fileName string) (parquet.ReaderAtSeeker, error) {
	fr, err := f.m.Open(fileName)
	if err != nil {
		return nil, errors.Errorf("failed to open file in memory fs mock: %v", err)
	}

	return fr, nil
}

func (f *fsMock) Stat(name string) (os.FileInfo, error) {
	return f.m.Stat(name) //nolint:wrapcheck
}

func (f *fsMock) FindFilesWithExtension(dir, ext string) ([]string, error) {
	fileInfo, err := f.m.Stat(dir)
	if os.IsNotExist(err) {
		return nil, errors.Errorf("directory %q does not exist", dir)
	}

	if !fileInfo.IsDir() {
		return nil, errors.Errorf("%q not a directory", dir)
	}

	var files []string

	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	err = afero.Walk(f.m, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.New(err.Error())
		}

		if !info.IsDir() && filepath.Ext(path) == ext {
			files = append(files, info.Name())
		}

		return nil
	})

	if err != nil {
		return nil, errors.WithMessagef(err, "failed to walk directory %q", dir)
	}

	return files, nil
}

func (f *fsMock) FindFilesWithPrefix(dir, prefix string) ([]string, error) {
	fileInfo, err := f.m.Stat(dir)
	if os.IsNotExist(err) {
		return nil, errors.Errorf("directory %q does not exist", dir)
	}

	if !fileInfo.IsDir() {
		return nil, errors.Errorf("%q not a directory", dir)
	}

	var files []string

	err = afero.Walk(f.m, dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.New(err.Error())
		}

		if !info.IsDir() && strings.HasPrefix(info.Name(), prefix) {
			files = append(files, info.Name())
		}

		return nil
	})

	if err != nil {
		return nil, errors.WithMessagef(err, "failed to walk directory %q", dir)
	}

	return files, nil
}
