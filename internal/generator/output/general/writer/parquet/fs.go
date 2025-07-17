package parquet

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/arrow-go/v18/parquet"
	"github.com/pkg/errors"

	"sdvg/internal/generator/common"
)

func NewFileSystem() FileSystem {
	return &fs{}
}

type fs struct{}

func (f *fs) NewFileWriter(fileName string) (io.WriteCloser, error) {
	fw, err := os.Create(fileName)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return fw, nil
}

func (f *fs) NewLocalFileReader(fileName string) (parquet.ReaderAtSeeker, error) {
	return os.Open(fileName) //nolint:wrapcheck
}

func (f *fs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name) //nolint:wrapcheck
}

func (f *fs) FindFilesWithExtension(dir, ext string) ([]string, error) {
	return common.WalkWithFilter(dir, func(e os.DirEntry) bool {
		return !e.IsDir() && filepath.Ext(e.Name()) == ext
	})
}

func (f *fs) FindFilesWithPrefix(dir, prefix string) ([]string, error) {
	return common.WalkWithFilter(dir, func(e os.DirEntry) bool {
		return !e.IsDir() && strings.HasPrefix(e.Name(), prefix)
	})
}
