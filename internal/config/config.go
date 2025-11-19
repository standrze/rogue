package config

import (
	"encoding/json"
	"os"
)

type LoggingConfig struct {
	SessionDir   string `json:"session_dir" mapstructure:"session_dir"`
	LogRequests  bool   `json:"log_requests" mapstructure:"log_requests"`
	LogResponses bool   `json:"log_responses" mapstructure:"log_responses"`
	LogHeaders   bool   `json:"log_headers" mapstructure:"log_headers"`
	LogBody      bool   `json:"log_body" mapstructure:"log_body"`
	MaxBodySize  int    `json:"max_body_size" mapstructure:"max_body_size"`
}

type CertificateConfig struct {
	AutoGenerate bool   `json:"auto_generate" mapstructure:"auto_generate"`
	Organization string `json:"organization" mapstructure:"organization"`
	CommonName   string `json:"common_name" mapstructure:"common_name"`
	ValidDays    int    `json:"valid_days" mapstructure:"valid_days"`
	CertPath     string `json:"cert_path" mapstructure:"cert_path"`
	KeyPath      string `json:"key_path" mapstructure:"key_path"`
}

type ProxyConfig struct {
	Port    int    `json:"port" mapstructure:"port"`
	Host    string `json:"host" mapstructure:"host"`
	Timeout int    `json:"timeout" mapstructure:"timeout"`
}

type Config struct {
	Proxy       ProxyConfig       `json:"proxy" mapstructure:"proxy"`
	Certificate CertificateConfig `json:"certificate" mapstructure:"certificate"`
	Logging     LoggingConfig     `json:"logging" mapstructure:"logging"`
}

func DefaultConfig() *Config {
	return &Config{
		Proxy: ProxyConfig{
			Port:    8080,
			Host:    "0.0.0.0",
			Timeout: 30,
		},
		Certificate: CertificateConfig{
			AutoGenerate: true,
			Organization: "Rogue Proxy",
			CommonName:   "Rogue CA",
			ValidDays:    365,
			CertPath:     "certs/ca.crt",
			KeyPath:      "certs/ca.key",
		},
		Logging: LoggingConfig{
			SessionDir:   "logs",
			LogRequests:  true,
			LogResponses: true,
			LogHeaders:   true,
			LogBody:      true,
			MaxBodySize:  1024 * 1024, // 1MB
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
