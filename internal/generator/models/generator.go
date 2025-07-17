package models

import (
	"bytes"
	"runtime"
	"time"

	"github.com/pkg/errors"

	"sdvg/internal/generator/common"
)

const (
	DefaultWorkersPerCPU = 4
)

// GenerationConfig type is used to describe config and model of generated data.
type GenerationConfig struct {
	WorkersCount int    `json:"workers_count" yaml:"workers_count"`
	BatchSize    uint64 `json:"batch_size"    yaml:"batch_size"`
	//nolint:lll
	RandomSeed     uint64            `backup:"true"           json:"random_seed"      yaml:"random_seed"` // only for backup
	RealRandomSeed uint64            `json:"-"                yaml:"-"`
	OutputConfig   *OutputConfig     `backup:"true"           json:"output"           yaml:"output"`
	Models         map[string]*Model `backup:"true"           json:"models"           yaml:"models"`
	ModelsToIgnore []string          `json:"models_to_ignore" yaml:"models_to_ignore"`
}

func (gc *GenerationConfig) ParseFromFile(path string) error {
	err := DecodeFile(path, gc)
	if err != nil {
		return errors.WithMessagef(err, "failed to parse generator config file %q", path)
	}

	err = gc.PostProcess()
	if err != nil {
		return err
	}

	return nil
}

func (gc *GenerationConfig) ParseFromYAML(data []byte) error {
	err := DecodeReader("yaml", bytes.NewReader(data), gc)
	if err != nil {
		return errors.WithMessage(err, "failed to parse YAML generator config")
	}

	err = gc.PostProcess()
	if err != nil {
		return err
	}

	return nil
}

func (gc *GenerationConfig) ParseFromJSON(data []byte) error {
	err := DecodeReader("json", bytes.NewReader(data), gc)
	if err != nil {
		return errors.WithMessage(err, "failed to parse JSON generator config")
	}

	err = gc.PostProcess()
	if err != nil {
		return err
	}

	return nil
}

func (gc *GenerationConfig) PostProcess() error {
	err := gc.Parse()
	if err != nil {
		return errors.WithMessage(err, "failed to parse generator config")
	}

	gc.FillDefaults()

	errs := gc.Validate()
	if len(errs) != 0 {
		return errors.Errorf("failed to validate generator config:\n%v", parseErrsToString(errs))
	}

	return nil
}

func (gc *GenerationConfig) Parse() error {
	gc.RealRandomSeed = gc.RandomSeed

	if err := gc.parseModels(); err != nil {
		return err
	}

	if err := gc.parseForeignKeys(); err != nil {
		return err
	}

	if err := gc.parseOutput(); err != nil {
		return err
	}

	return nil
}

func (gc *GenerationConfig) parseModels() error {
	if len(gc.Models) == 0 {
		return errors.New("no model to generate")
	}

	for modelName := range gc.Models {
		gc.Models[modelName].Name = modelName

		err := gc.Models[modelName].Parse()
		if err != nil {
			return errors.WithMessagef(err, "models[%s]", modelName)
		}
	}

	return nil
}

func (gc *GenerationConfig) parseForeignKeys() error {
	columnsMap := make(map[string]*Column)

	for modelName, model := range gc.Models {
		for _, column := range model.Columns {
			columnsMap[common.GetKey(modelName, column.Name)] = column
		}
	}

	for ck, column := range columnsMap {
		if column.ForeignKey != "" {
			var ok bool

			column.ForeignKeyColumn, ok = columnsMap[column.ForeignKey]
			if !ok {
				return errors.Errorf("incorrect foreign key for %q: unknown column %q", ck, column.ForeignKey)
			}

			if column.ForeignKeyColumn.ForeignKey != "" {
				return errors.Errorf("incorrect foreign key for %q: %q is a foreign key", ck, column.ForeignKeyColumn.ForeignKey)
			}
		}
	}

	return nil
}

func (gc *GenerationConfig) parseOutput() error {
	if gc.OutputConfig == nil {
		gc.OutputConfig = &OutputConfig{}
	}

	if err := gc.OutputConfig.Parse(); err != nil {
		return err
	}

	return nil
}

func (gc *GenerationConfig) FillDefaults() {
	if gc.WorkersCount == 0 {
		gc.WorkersCount = runtime.GOMAXPROCS(0) * DefaultWorkersPerCPU
	}

	if gc.BatchSize == 0 {
		gc.BatchSize = 1000
	}

	if gc.RandomSeed == 0 {
		gc.RandomSeed = uint64(time.Now().UnixNano())
	}

	gc.fillDefaultsModels()

	gc.fillDefaultsOutput()
}

func (gc *GenerationConfig) fillDefaultsModels() {
	for _, model := range gc.Models {
		model.FillDefaults()
	}
}

func (gc *GenerationConfig) fillDefaultsOutput() {
	gc.OutputConfig.FillDefaults()
}

func (gc *GenerationConfig) Validate() []error {
	var errs []error

	// Validate direct properties
	if gc.WorkersCount <= 0 {
		errs = append(errs, errors.Errorf("workers count should be grater than 0, got %v", gc.WorkersCount))
	}

	if modelsErrs := gc.validateModels(); len(modelsErrs) != 0 {
		errs = append(errs, modelsErrs...)
	}

	if outputErrs := gc.OutputConfig.Validate(); len(outputErrs) != 0 {
		errs = append(errs, errors.New("output config:"))
		errs = append(errs, outputErrs...)
	}

	return errs
}

func (gc *GenerationConfig) validateModels() []error {
	var errs []error

	for modelName, model := range gc.Models {
		modelErrs := model.Validate()
		if len(modelErrs) != 0 {
			errs = append(errs, errors.Errorf("models[%s]:", modelName))
			errs = append(errs, modelErrs...)
		}
	}

	for _, modelName := range gc.ModelsToIgnore {
		if _, ok := gc.Models[modelName]; !ok {
			errs = append(errs, errors.Errorf("unknown model to ignore %q", modelName))
		}
	}

	if len(gc.ModelsToIgnore) == len(gc.Models) {
		errs = append(errs, errors.Errorf("all models are marked as ignored"))
	}

	return errs
}
