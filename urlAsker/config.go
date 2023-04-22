package main

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	FileName   string        `yaml:"fileName"`
	BufferSize int           `yaml:"bufferSize"`
	Workers    int           `yaml:"workers"`
	Timeout    time.Duration `yaml:"timeout"`
	Retries    int           `yaml:"retries"`
	Urls       []string      `yaml:"urls"`
}

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatalf("can't close file: %v", err)
		}
	}()

	d := yaml.NewDecoder(f)
	if err = d.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
