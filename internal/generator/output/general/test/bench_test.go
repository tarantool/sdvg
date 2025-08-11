package test

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general"
	"github.com/tarantool/sdvg/internal/generator/output/mock"
	uc "github.com/tarantool/sdvg/internal/generator/usecase"
	usecase "github.com/tarantool/sdvg/internal/generator/usecase/general"
)

const (
	BatchSize                   = 1000
	PartitionColumnUniqueValues = 100
)

var (
	BatchByModelName = make(map[string][]*models.DataRow, len(GetModels(PartitionColumnUniqueValues, BatchSize)))
)

func GetModels(partitionColumnUniqueValues uint64, rowsCount uint64) map[string]*models.Model {
	var (
		from int64 = 1
		to   int64 = math.MaxInt32

		fromFloat = float64(1)
		toFloat   = math.MaxFloat32
	)

	return map[string]*models.Model{
		"integers": {
			Name:        "integers",
			RowsCount:   rowsCount,
			RowsPerFile: rowsCount,
			Columns: []*models.Column{
				{
					Name: "integer_32",
					Type: "integer",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnIntegerParams{
							BitWidth: 32,
						},
						DistinctCount:   partitionColumnUniqueValues,
						RangePercentage: 1,
					}},
				},
				{
					Name: "integer_64",
					Type: "integer",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnIntegerParams{
							BitWidth: 64,
							FromPtr:  &from,
							ToPtr:    &to,
						},
						DistinctPercentage: 1,
						RangePercentage:    1,
					}},
				},
			},
			PartitionColumns: []*models.PartitionColumn{
				{
					Name:          "integer_32",
					WriteToOutput: true,
				},
			},
		},
		"floats": {
			Name:        "floats",
			RowsCount:   rowsCount,
			RowsPerFile: rowsCount,
			Columns: []*models.Column{
				{
					Name: "float_32",
					Type: "float",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnFloatParams{
							BitWidth: 32,
						},
						DistinctPercentage: 1,
						RangePercentage:    1,
					}},
				},
				{
					Name: "float_64",
					Type: "float",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnFloatParams{
							BitWidth: 64,
							FromPtr:  &fromFloat,
							ToPtr:    &toFloat,
						},
						DistinctCount:   partitionColumnUniqueValues,
						RangePercentage: 1,
					}},
				},
			},
			PartitionColumns: []*models.PartitionColumn{
				{
					Name:          "float_64",
					WriteToOutput: true,
				},
			},
		},
		"strings": {
			Name:        "strings",
			RowsCount:   rowsCount,
			RowsPerFile: rowsCount,
			Columns: []*models.Column{
				{
					Name: "string",
					Type: "string",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnStringParams{
							Locale: "en", MinLength: 32, MaxLength: 32,
						},
						DistinctCount:   partitionColumnUniqueValues,
						RangePercentage: 1,
					}},
				},
				{
					Name: "uuid",
					Type: "uuid",
					Ranges: []*models.Params{{
						DistinctPercentage: 1,
						RangePercentage:    1,
					}},
				},
			},
			PartitionColumns: []*models.PartitionColumn{
				{
					Name:          "string",
					WriteToOutput: true,
				},
			},
		},
		"datetime": {
			Name:        "datetime",
			RowsCount:   rowsCount,
			RowsPerFile: rowsCount,
			Columns: []*models.Column{
				{
					Name: "created_dt",
					Type: "datetime",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnDateTimeParams{
							From: time.Date(1995, time.February, 17, 0, 0, 0, 0, time.UTC),
							To:   time.Date(2002, time.February, 27, 0, 0, 0, 0, time.UTC),
						},
						DistinctPercentage: 1,
						RangePercentage:    1,
					}},
				},
				{
					Name: "started_dt",
					Type: "datetime",
					Ranges: []*models.Params{{
						TypeParams: &models.ColumnDateTimeParams{
							From: time.Date(2002, time.February, 27, 0, 0, 0, 0, time.UTC),
							To:   time.Date(2010, time.March, 20, 0, 0, 0, 0, time.UTC),
						},
						DistinctCount:   partitionColumnUniqueValues,
						RangePercentage: 1,
					}},
				},
			},
			PartitionColumns: []*models.PartitionColumn{
				{
					Name:          "started_dt",
					WriteToOutput: true,
				},
			},
		},
	}
}

func avgRowSizeBytes(model *models.Model) (int64, error) {
	totalSize := int64(0)

	for _, column := range model.Columns {
		for _, r := range column.Ranges {
			switch r.ColumnType {
			case "integer":
				totalSize += int64(r.RangePercentage * float64(r.IntegerParams.BitWidth) / 8)
			case "float":
				totalSize += int64(r.RangePercentage * float64(r.FloatParams.BitWidth) / 8)
			case "string":
				avgStrSize := float64(r.StringParams.MaxLength+r.StringParams.MinLength) / 2
				totalSize += int64(r.RangePercentage * avgStrSize)
			case "datetime":
				totalSize += 16
			case "uuid":
				totalSize += 16
			default:
				return 0, errors.Errorf("unexpected column type '%s'", column.Type)
			}
		}
	}

	return totalSize, nil
}

func init() {
	var err error

	BatchByModelName, err = generateBatches(GetBatchGenCfg())
	if err != nil {
		log.Fatalf("failed to generate data for output benchmark tests: %s", err)
	}
}

func GetBatchGenCfg() *models.GenerationConfig {
	allModels := GetModels(PartitionColumnUniqueValues, BatchSize)

	genCfg := &models.GenerationConfig{
		BatchSize:    BatchSize,
		RandomSeed:   69,
		OutputConfig: nil,
		Models:       allModels,
	}

	if err := genCfg.Parse(); err != nil {
		log.Fatalf("failed parse config: %s", err)
	}

	genCfg.FillDefaults()

	if err := genCfg.Validate(); err != nil {
		log.Fatalf("failed validate config: %s", err)
	}

	return genCfg
}

func generateBatches(cfg *models.GenerationConfig) (map[string][]*models.DataRow, error) {
	batchByModelName := make(map[string][]*models.DataRow, len(cfg.Models))

	mutex := &sync.Mutex{}
	out := mock.NewOutput(func(_ context.Context, modelName string, rows []*models.DataRow) error {
		mutex.Lock()
		batchByModelName[modelName] = rows
		mutex.Unlock()

		return nil
	})

	useCase := usecase.NewUseCase(usecase.UseCaseConfig{})

	taskID, err := useCase.CreateTask(
		context.Background(),
		uc.TaskConfig{
			GenerationConfig: cfg,
			Output:           out,
		},
	)

	if err != nil {
		return nil, err
	}

	if err = useCase.WaitResult(taskID); err != nil {
		return nil, err
	}

	return batchByModelName, nil
}

func runModelsBenches(
	b *testing.B,
	genCfg *models.GenerationConfig,
) {
	b.Helper()

	bytesPerIteration := int64(0)

	for modelName := range genCfg.Models {
		rowSize, err := avgRowSizeBytes(genCfg.Models[modelName])
		require.NoError(b, err)

		bytesPerIteration += rowSize
	}

	b.Run("CI/"+"cpu", func(b *testing.B) {
		b.Helper()
		b.SetBytes(bytesPerIteration)

		copyCfg := *genCfg
		SetOutputParams(&copyCfg, uint64(b.N))

		out := general.NewOutput(&copyCfg, false, true, nil)
		require.NoError(b, out.Setup())

		b.ResetTimer()

		var wg = &sync.WaitGroup{}

		for modelName := range genCfg.Models {
			wg.Add(1)

			go func(b *testing.B, modelName string, batch []*models.DataRow) {
				b.Helper()

				defer wg.Done()

				for i := 0; i < b.N; i += BatchSize {
					batchSize := min(b.N-i, BatchSize)
					if err := out.HandleRowsBatch(context.Background(), modelName, batch[:batchSize]); err != nil {
						b.Errorf("failed write batch for model '%s'", modelName)
					}
				}
			}(b, modelName, BatchByModelName[modelName])
		}

		wg.Wait()

		require.NoError(b, out.Teardown())

		b.StopTimer()
		teardown(b)

		numberOfColumns := float64(0)
		for _, model := range copyCfg.Models {
			numberOfColumns += float64(len(model.Columns))
		}

		b.ReportMetric(float64(b.N*len(copyCfg.Models))/b.Elapsed().Seconds(), "rows/s")
		b.ReportMetric(float64(b.N)*numberOfColumns/b.Elapsed().Seconds(), "values/s")
	})
}

func SetOutputParams(cfg *models.GenerationConfig, rowsCount uint64) {
	for _, model := range cfg.Models {
		model.RowsCount = rowsCount
		model.RowsPerFile = model.RowsCount
	}
}

func teardown(b *testing.B) {
	b.Helper()

	err := os.RemoveAll(models.DefaultOutputDir)
	require.NoError(b, err)
}

func BenchmarkPartitioning(b *testing.B) {
	genCfg := GetBatchGenCfg()

	genCfg.OutputConfig = &models.OutputConfig{
		Type:               "devnull",
		Dir:                b.TempDir(),
		CheckpointInterval: 5 * time.Second,
		DevNullParams: &models.DevNullConfig{
			Handler: func(_ *models.DataRow, _ string) error {
				return nil
			},
		},
	}

	runModelsBenches(b, genCfg)
}

func RemovePartitions(cfg *models.GenerationConfig) {
	for _, model := range cfg.Models {
		model.PartitionColumns = nil
	}
}

func BenchmarkCSVWriter(b *testing.B) {
	genCfg := GetBatchGenCfg()

	genCfg.OutputConfig = &models.OutputConfig{
		Type:               "csv",
		Dir:                b.TempDir(),
		CheckpointInterval: 5 * time.Second,
		CSVParams: &models.CSVConfig{
			WithoutHeaders: false,
			Delimiter:      ",",
			DatetimeFormat: "2002-02-27T15:04:05Z07:00",
			FloatPrecision: 2,
		},
	}

	RemovePartitions(genCfg)
	runModelsBenches(b, genCfg)
}

func BenchmarkParquetWriter(b *testing.B) {
	genCfg := GetBatchGenCfg()

	genCfg.OutputConfig = &models.OutputConfig{
		Type:               "parquet",
		Dir:                b.TempDir(),
		CheckpointInterval: 5 * time.Second,
		ParquetParams: &models.ParquetConfig{
			CompressionCodec: "UNCOMPRESSED",
			FloatPrecision:   2,
			DateTimeFormat:   models.ParquetDateTimeMillisFormat,
		},
	}

	for _, model := range genCfg.Models {
		for _, column := range model.Columns {
			column.ParquetParams = &models.ColumnParquetParams{
				Encoding: "Plain",
			}
		}
	}

	RemovePartitions(genCfg)
	runModelsBenches(b, genCfg)
}

func BenchmarkHTTPWriter(b *testing.B) {
	server := initMockServer()
	defer server.Close()

	genCfg := GetBatchGenCfg()

	genCfg.OutputConfig = &models.OutputConfig{
		Type:               "http",
		Dir:                b.TempDir(),
		CheckpointInterval: 5 * time.Second,
		HTTPParams: &models.HTTPParams{
			Endpoint:     server.URL + "/test",
			Timeout:      time.Second * 30,
			BatchSize:    1000,
			WorkersCount: 1,
		},
	}

	RemovePartitions(genCfg)
	runModelsBenches(b, genCfg)
}

func BenchmarkTCSWriter(b *testing.B) {
	server := initMockServer()
	defer server.Close()

	genCfg := GetBatchGenCfg()

	genCfg.OutputConfig = &models.OutputConfig{
		Type:               "tcs",
		Dir:                b.TempDir(),
		CheckpointInterval: 5 * time.Second,
		TCSParams: &models.TCSConfig{
			HTTPParams: models.HTTPParams{
				Endpoint:     server.URL + "/test",
				Timeout:      time.Second * 30,
				BatchSize:    1000,
				WorkersCount: 1,
			},
		},
	}

	RemovePartitions(genCfg)
	runModelsBenches(b, genCfg)
}

func initMockServer() *httptest.Server {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, er := w.Write([]byte(`{"message": "yet another fail 💀"}`))

			if er != nil {
				slog.Warn(fmt.Sprintf("failed reply to client: %s", er))
			}

			return
		}

		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)

		_, er := w.Write([]byte(`{"message": "yet another success 😎"}`))

		if er != nil {
			slog.Warn(fmt.Sprintf("failed reply to client: %s", er))
		}
	}))

	return mockServer
}
