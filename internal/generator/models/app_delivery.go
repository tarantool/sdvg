package models

import (
	"time"

	"github.com/pkg/errors"
)

// HTTPConfig type used to describe delivery config for http implementation.
type HTTPConfig struct {
	ListenAddress string        `json:"listen_address" yaml:"listen_address"`
	ReadTimeout   time.Duration `json:"read_timeout"   yaml:"read_timeout"`
	WriteTimeout  time.Duration `json:"write_timeout"  yaml:"write_timeout"`
	IdleTimeout   time.Duration `json:"idle_timeout"   yaml:"idle_timeout"`
}

func (c *HTTPConfig) FillDefaults() {
	if c.ListenAddress == "" {
		c.ListenAddress = ":8080"
	}

	if c.ReadTimeout == 0 {
		c.ReadTimeout = time.Minute
	}

	if c.WriteTimeout == 0 {
		c.WriteTimeout = time.Minute
	}

	if c.IdleTimeout == 0 {
		c.IdleTimeout = time.Minute
	}
}

func (c *HTTPConfig) Validate() []error {
	var errs []error

	if c.ReadTimeout < 0 {
		errs = append(errs, errors.Errorf("read timeout should be grater than 0, got %v", c.ReadTimeout))
	}

	if c.WriteTimeout < 0 {
		errs = append(errs, errors.Errorf("write timeout should be grater than 0, got %v", c.WriteTimeout))
	}

	if c.IdleTimeout < 0 {
		errs = append(errs, errors.Errorf("idle timeout should be grater than 0, got %v", c.IdleTimeout))
	}

	return errs
}
