package test

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
	outputGeneral "github.com/tarantool/sdvg/internal/generator/output/general"
	"github.com/tarantool/sdvg/internal/generator/output/general/writer"
	outputCsv "github.com/tarantool/sdvg/internal/generator/output/general/writer/csv"
	outputParquet "github.com/tarantool/sdvg/internal/generator/output/general/writer/parquet"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	useCaseGeneral "github.com/tarantool/sdvg/internal/generator/usecase/general"
)

const (
	configFileName  = "config.yml"
	configFilePerm  = 0644
	twoModelsConfig = `
models:
  model1:
    rows_count: 15
    rows_per_file: 4
    columns:
      - name: testColumn
        type: string
        ranges:
          - values: ["a", "b", "c"]
            range_percentage: 0.3
          - values: ["1", "2", "3"]
            range_percentage: 0.4
          - values: ["-", "+", "?"]
            range_percentage: 0.3
  model2:
    rows_count: 36
    rows_per_file: 7
    columns:
      - name: testColumn
        type: integer
        ranges:
          - type_params:
              from: 22
            range_percentage: 0.5
          - type_params:
              to: 5
`
	oneModelConfig = `
models:
  model2:
    rows_count: 36
    rows_per_file: 7
    columns:
      - name: testColumn
        type: integer
        ranges:
          - type_params:
              from: 22
            range_percentage: 0.5
          - type_params:
              to: 5
`
)

//nolint:cyclop
func TestContinueGeneration(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")

	// Write models config

	configPath := filepath.Join(tempDir, configFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(twoModelsConfig), configFilePerm))

	uc := useCaseGeneral.NewUseCase(useCaseGeneral.UseCaseConfig{})
	require.NoError(t, uc.Setup())

	// Parse config

	cfg := &models.GenerationConfig{}
	require.NoError(t, cfg.ParseFromFile(configPath))
	cfg.OutputConfig.Dir = outputDir

	rowsCountByModel := make(map[string]uint64, len(cfg.Models))
	for _, model := range cfg.Models {
		rowsCountByModel[model.Name] = model.GenerateTo - model.GenerateFrom
	}

	// Generate expected data

	require.NoError(t, generate(t, cfg, uc, false, true))

	expectedFilesData := make(map[string][][]string)

	for _, model := range cfg.Models {
		filesCount := int(math.Ceil(float64(rowsCountByModel[model.Name]) / float64(model.RowsPerFile)))

		for i := range filesCount {
			filePath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.csv", model.Name, i))
			data := readCSVFile(t, filePath)

			expectedFilesData[filePath] = data
		}
	}

	require.NoError(t, os.RemoveAll(outputDir))

	// Write half of the data

	for _, model := range cfg.Models {
		model.GenerateTo = model.RowsCount / 2
	}

	require.NoError(t, generate(t, cfg, uc, false, true))

	for _, model := range cfg.Models {
		filesCount := int(math.Ceil(float64(model.GenerateTo-model.GenerateFrom) / float64(model.RowsPerFile)))

		for i := range filesCount {
			filePath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.csv", model.Name, i))
			expectedData := expectedFilesData[filePath]

			data := readCSVFile(t, filePath)

			expectedLen := int(model.RowsPerFile)

			if i == filesCount-1 {
				expectedLen = int(model.GenerateTo - model.GenerateFrom - model.RowsPerFile*uint64(i))
			}

			if !cfg.OutputConfig.CSVParams.WithoutHeaders {
				expectedLen++
			}

			require.Len(t, data, expectedLen)

			for j := range data {
				require.Equal(t, expectedData[j], data[j])
			}
		}
	}

	// ContinueGeneration
	cfg = &models.GenerationConfig{}
	require.NoError(t, cfg.ParseFromFile(configPath))
	cfg.OutputConfig.Dir = outputDir

	require.NoError(t, generate(t, cfg, uc, true, true))

	for _, model := range cfg.Models {
		filesCount := math.Ceil(float64(rowsCountByModel[model.Name]) / float64(model.RowsPerFile))

		for i := range int(filesCount) {
			filePath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.csv", model.Name, i))
			data := readCSVFile(t, filePath)

			expectedData := expectedFilesData[filePath]
			require.Equal(t, len(data), len(expectedData))

			for j := range data {
				require.Equal(t, expectedData[j], data[j])
			}
		}
	}
}

func TestForceGeneration(t *testing.T) {
	testCases := []struct {
		name                string
		err                 error
		forceGeneration     bool
		createModelDirParam bool
	}{
		{
			"expected error, create_model_dir: false",
			errors.New(`conflict files found in output dir:
cause: SDVG metadata file
	- output/backup.json
	- output/model2_checkpoint.json
cause: files with old models data
	- output/model2_0.csv
	- output/model2_1.csv
	- output/model2_2.csv
	- output/model2_3.csv
	- output/model2_4.csv
`),
			false,
			false,
		},
		{
			"expected error, create_model_dir: true",
			errors.New(`conflict files found in output dir:
cause: SDVG metadata file
	- output/backup.json
cause: dir for model is not empty
	- output/model2
`),
			false,
			true,
		},
		{
			"force generation, create_model_dir: false",
			nil,
			true,
			false,
		},
		{
			"force generation, create_model_dir: true",
			nil,
			true,
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write models config
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, configFileName)
			require.NoError(t, os.WriteFile(configPath, []byte(oneModelConfig), configFilePerm))

			uc := useCaseGeneral.NewUseCase(useCaseGeneral.UseCaseConfig{})
			require.NoError(t, uc.Setup())

			// Parse config

			cfg := &models.GenerationConfig{}

			require.NoError(t, cfg.ParseFromFile(configPath))

			cfg.OutputConfig.CreateModelDir = tc.createModelDirParam

			// Generate data in empty output dir

			require.NoError(t, generate(t, cfg, uc, false, false))

			// Try to init new output with conflicts
			out := outputGeneral.NewOutput(cfg, false, tc.forceGeneration)

			err := out.Setup()
			if tc.err != nil {
				require.Error(t, err)
				require.Equal(t, tc.err.Error(), err.Error())
			}

			// Check that all conflict files have been deleted
			if tc.createModelDirParam {
				fileNames, err := common.WalkWithFilter(tmpDir, func(entry os.DirEntry) bool {
					return entry.Name() != configFileName
				})

				require.Empty(t, fileNames, 0, "there should be no files after force generation")
				require.NoError(t, err, "failed to walk tmpdir: %v", tmpDir)
			}

			require.NoError(t, os.RemoveAll(models.DefaultOutputDir))
		})
	}
}

// TestWriterInitTeardown tests if Teardown works properly right after Init
func TestWriterInitTeardown(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name   string
		writer writer.Writer
	}{
		{
			"csv",
			outputCsv.NewWriter(
				context.TODO(),
				nil,
				//w.config.CSVParams,
				&models.CSVConfig{
					FloatPrecision: 1,
					DatetimeFormat: "2006-01-02T15:04:05Z07:00",
					Delimiter:      ",",
					WithoutHeaders: false,
				},
				nil,
				tmpDir,
				false,
				make(chan<- uint64),
			),
		},
		{
			"parquet",
			outputParquet.NewWriter(
				&models.Model{Columns: make([]*models.Column, 0)},
				&models.ParquetConfig{
					"UNCOMPRESSED",
					2,
					models.ParquetDateTimeMillisFormat,
				},
				nil,
				outputParquet.NewFileSystem(),
				tmpDir,
				false,
				make(chan<- uint64),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.writer.Init())
			require.NoError(t, tc.writer.Teardown())
		})
	}
}

//nolint:lll
func generate(t *testing.T, cfg *models.GenerationConfig, uc usecase.UseCase, continueGeneration, forceGeneration bool) error {
	t.Helper()

	out := outputGeneral.NewOutput(cfg, continueGeneration, forceGeneration)

	taskID, err := uc.CreateTask(context.Background(), usecase.TaskConfig{
		GenerationConfig:   cfg,
		Output:             out,
		ContinueGeneration: continueGeneration,
	})
	if err != nil {
		return err
	}

	require.NoError(t, uc.WaitResult(taskID))
	require.NoError(t, uc.Teardown())

	return nil
}

func readCSVFile(t *testing.T, path string) [][]string {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)

	records, err := csv.NewReader(file).ReadAll()
	require.NoError(t, err)
	require.NoError(t, file.Close())

	return records
}
