package backup

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output"
	"github.com/tarantool/sdvg/internal/generator/output/general"
)

func TestHandleBackup(t *testing.T) {
	tempDir := t.TempDir()

	baseCfg := &models.GenerationConfig{
		Models: map[string]*models.Model{
			"test_model": {
				RowsCount: 1,
			},
		},
	}

	require.NoError(t, baseCfg.PostProcess())

	out := general.NewOutput(
		&models.GenerationConfig{
			OutputConfig: &models.OutputConfig{
				Dir: tempDir,
			},
		},
		false,
		false,
		nil,
	)

	type testCase struct {
		name               string
		wantErr            bool
		realRandomSeed     uint64
		expectedRandomSeed uint64
		backupContent      string
	}

	testCases := []testCase{
		{
			name:               "Real random seed equals to zero",
			wantErr:            false,
			realRandomSeed:     0,
			expectedRandomSeed: 123456,
			backupContent: `{ 
	"random_seed": 123456,
	"models": {
		"test_model": {
			"rows_count": 1
		}
	} 
}`,
		},
		{
			name:               "Real random seed not equals to zero and correct",
			wantErr:            false,
			realRandomSeed:     123,
			expectedRandomSeed: 123,
			backupContent: `{ 
	"random_seed": 123,
	"models": {
		"test_model": {
			"rows_count": 1
		}
	} 
}`,
		},
		{
			name:           "Real random seed not equals to zero and not correct",
			wantErr:        true,
			realRandomSeed: 123,
			backupContent: `{ 
	"random_seed": 123456,
	"models": {
		"test_model": {
			"rows_count": 1
		}
	} 
}`,
		},
		{
			name:    "Random seed is not exists",
			wantErr: true,
			backupContent: `{
	"models": {
		"test_model": {
			"rows_count": 1
		}
	} 
}`,
		},
		{
			name:    "Random seed has unsupported type",
			wantErr: true,
			backupContent: `{ 
	"random_seed": true,
	"models": {
		"test_model": {
			"rows_count": 1
		}
	} 
}`,
		},
		{
			name:           "Actual config have diff from backup config",
			wantErr:        true,
			realRandomSeed: 0,
			backupContent: `{ 
	"random_seed": 123456,
	"models": {
		"test_model": {
			"rows_count": 2
		}
	} 
}`,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, output.BackupName), []byte(tc.backupContent), 0644))

		baseCfg.RandomSeed = tc.realRandomSeed
		baseCfg.RealRandomSeed = tc.realRandomSeed

		err := handleBackup(baseCfg, out)
		require.Equal(t, tc.wantErr, err != nil)

		if !tc.wantErr {
			require.Equal(t, tc.expectedRandomSeed, baseCfg.RandomSeed)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestHandleCheckpoint(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()

	generateTo := uint64(math.MaxUint64)

	cfg := &models.GenerationConfig{
		OutputConfig: &models.OutputConfig{
			Dir:                tempDir,
			CheckpointInterval: time.Minute,
		},
		Models: map[string]*models.Model{
			"model1": {
				Name:         "model1",
				GenerateFrom: 30,
				GenerateTo:   generateTo,
			},
			"model2": {
				Name:         "model2",
				GenerateFrom: 0,
				GenerateTo:   generateTo,
			},
			"model3": {
				Name:         "model3",
				GenerateFrom: 25000,
				GenerateTo:   generateTo,
			},
			"model4": {
				Name:         "model4",
				GenerateFrom: 1356,
				GenerateTo:   generateTo,
			},
		},
	}

	checkpoints := map[string]uint64{
		"model1": 542,
		"model2": 56975,
		"model3": 124356,
		"model4": 954,
	}

	out := general.NewOutput(cfg, false, false, nil)
	require.NoError(t, out.Setup())

	for modelName, generateFrom := range checkpoints {
		require.NoError(t, os.WriteFile(
			filepath.Join(tempDir, modelName+output.CheckpointSuffix),
			[]byte(fmt.Sprintf(`{ "saved_rows": %d }`, generateFrom)),
			0644,
		))
	}

	require.NoError(t, handleCheckpoint(cfg, out))

	for modelName, generateFrom := range checkpoints {
		require.Equal(t, generateFrom, cfg.Models[modelName].GenerateFrom)
	}
}
