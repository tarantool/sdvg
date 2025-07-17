package models

import (
	"slices"

	"github.com/pkg/errors"
)

// AppConfig type is used to describe application config.
type AppConfig struct {
	LogFormat  string     `json:"log_format" yaml:"log_format"`
	HTTPConfig HTTPConfig `json:"http"       yaml:"http"`
	OpenAI     OpenAI     `json:"open_ai"    yaml:"open_ai"`
}

func (m *AppConfig) ParseFromFile(path string) error {
	if path != "" {
		err := DecodeFile(path, m)
		if err != nil {
			return errors.WithMessagef(err, "failed to parse app config file %q", path)
		}
	}

	err := m.PostProcess()
	if err != nil {
		return errors.WithMessagef(err, "failed to post process app config file %q", path)
	}

	return nil
}

func (m *AppConfig) PostProcess() error {
	m.FillDefaults()

	errs := m.Validate()
	if len(errs) != 0 {
		return errors.Errorf("failed to validate app config:\n%v", parseErrsToString(errs))
	}

	return nil
}

func (m *AppConfig) FillDefaults() {
	if m.LogFormat == "" {
		m.LogFormat = "text"
	}

	m.HTTPConfig.FillDefaults()
}

func (m *AppConfig) Validate() []error {
	var errs []error

	if !slices.Contains([]string{"text", "json"}, m.LogFormat) {
		errs = append(errs, errors.Errorf("unknown log format: %s", m.LogFormat))
	}

	httpParamsErrs := m.HTTPConfig.Validate()
	if len(httpParamsErrs) != 0 {
		errs = append(errs, errors.New("failed to validate HTTP configuration:"))
		errs = append(errs, httpParamsErrs...)
	}

	return errs
}
