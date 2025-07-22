package general

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/output"
)

const filesToShow = 5

var (
	ConflictFilesWithOldModelsData = "files with old models data"
	ConflictMetadataFile           = "SDVG metadata file"
	ConflictPossiblePartitionDirs  = "dirs with '=' symbol, possible partition conflict"
	ConflictModelDirNotEmpty       = "dir for model is not empty"
)

func handleConflicts(conflicts map[string][]string, forceGeneration bool) error {
	var err error

	for _, filePaths := range conflicts {
		if len(filePaths) == 0 {
			continue
		}

		if forceGeneration {
			err = deleteAllFiles(filePaths)
			if err != nil {
				return errors.WithMessagef(err, "failed to delete conflict files: %s", filePaths)
			}

			return nil
		} else {
			conflictsAsString := formatConflicts(conflicts)

			return errors.New(conflictsAsString)
		}
	}

	return nil
}

// checkDirForModel checks for existing files or directories that may conflict with model output.
func checkDirForModel(dir, model, extensionType string, createModelDir bool) (map[string][]string, error) {
	conflicts := make(map[string][]string) // key is cause, value is slice of file names

	if createModelDir {
		dir = filepath.Join(dir, model)

		empty, err := isEmpty(dir)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to check for empty model directory: %v", dir)
		}

		if !empty {
			conflicts[ConflictModelDirNotEmpty] = []string{dir}
		}

		return conflicts, nil
	}

	conflictFiles, err := assertNoMatches(dir,
		func(e os.DirEntry) bool {
			return !e.IsDir() && strings.HasPrefix(e.Name(), model) && strings.HasSuffix(e.Name(), extensionType)
		},
	)
	if err != nil {
		return nil, err
	}

	conflicts[ConflictFilesWithOldModelsData] = conflictFiles

	checkpointFileName := model + output.CheckpointSuffix

	checkpointPath := filepath.Join(dir, checkpointFileName)

	_, err = os.Stat(checkpointPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.WithMessagef(errors.New(err.Error()), "failed to stat checkpoint file: %s", checkpointPath)
	}

	if !os.IsNotExist(err) {
		conflicts[ConflictMetadataFile] = []string{checkpointPath}
	}

	return conflicts, nil
}

// isEmpty returns error if provided filepath is not a directory.
func isEmpty(filepath string) (bool, error) {
	f, err := os.Open(filepath)
	if os.IsNotExist(err) {
		return true, nil
	}

	if err != nil {
		return false, errors.WithMessagef(errors.New(err.Error()), "failed to open file: %s", filepath)
	}

	defer f.Close()

	_, err = f.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}

	if err != nil {
		return false, errors.WithMessagef(errors.New(err.Error()), "failed to read dir names")
	}

	return false, nil
}

// formatConflicts makes pretty string from conflicts map.
func formatConflicts(conflicts map[string][]string) string {
	causes := make([]string, 0, len(conflicts))
	for cause := range conflicts {
		causes = append(causes, cause)
	}

	slices.Sort(causes)

	var sb strings.Builder

	sb.WriteString("conflict files found in output dir:\n")

	for _, cause := range causes {
		files := conflicts[cause]
		if len(files) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("cause: %s\n", cause))

		if len(files) > filesToShow {
			files = files[:filesToShow]
		}

		for _, file := range files {
			sb.WriteString(fmt.Sprintf("\t- %s\n", file))
		}
	}

	return sb.String()
}

func checkBackupFile(conflicts map[string][]string, backupFilePath string) error {
	_, err := os.Stat(backupFilePath)
	if err != nil && !os.IsNotExist(err) {
		return errors.WithMessagef(errors.New(err.Error()), "failed to stat backup file %q", backupFilePath)
	}

	if !os.IsNotExist(err) {
		conflicts[ConflictMetadataFile] = append(conflicts[ConflictMetadataFile], backupFilePath)
	}

	return nil
}

func checkPossiblePartitionDirs(conflicts map[string][]string, dir string) error {
	possiblePartitionDirs, err := assertNoMatches(dir,
		func(e os.DirEntry) bool {
			return e.IsDir() && strings.Contains(e.Name(), "=")
		})
	if err != nil {
		return errors.WithMessagef(errors.New(err.Error()), "failed to check for possible partition dirs, dir: %v", dir)
	}

	conflicts[ConflictPossiblePartitionDirs] = append(conflicts[ConflictPossiblePartitionDirs], possiblePartitionDirs...)

	return nil
}

// assertNoMatches is a helper to ensure no matching entries exist under dir.
func assertNoMatches(dir string, matchFn func(entry os.DirEntry) bool) ([]string, error) {
	items, err := common.WalkWithFilter(dir, matchFn)
	if errors.Is(err, common.ErrDirNotExists) {
		return nil, nil
	}

	for i := range items {
		items[i] = filepath.Join(dir, items[i])
	}

	return items, err
}

func deleteAllFiles(filePaths []string) error {
	for _, filePath := range filePaths {
		if err := os.RemoveAll(filePath); err != nil {
			return errors.WithMessagef(errors.New(err.Error()), "failed to remove file: %s", filePath)
		}
	}

	return nil
}
