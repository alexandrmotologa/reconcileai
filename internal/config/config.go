package config

import (
	"encoding/json"
	"os"
)

// ToleranceConfig sets operational bounds for monetary and volume variances.
type ToleranceConfig struct {
	MaxPriceVariancePercent  float64 `json:"max_price_variance_percent"`  // e.g. 0.5 (0.5%)
	MaxPriceVarianceAbsolute float64 `json:"max_price_variance_absolute"` // e.g. 0.05 ($0.05)
	MaxQuantityVariance      float64 `json:"max_quantity_variance"`       // e.g. 0.0
	MaxTaxVarianceAbsolute   float64 `json:"max_tax_variance_absolute"`    // e.g. 0.02
}

// MatchingConfig configures bipartite semantic alignment parameters.
type MatchingConfig struct {
	MinDescriptionSimilarity float64 `json:"min_description_similarity"` // e.g. 0.65
	EnableUOMConversion      bool    `json:"enable_uom_conversion"`
	AllowFuzzySKUFallback    bool    `json:"allow_fuzzy_sku_fallback"`
}

// ERPConfig specifies export preferences.
type ERPConfig struct {
	DefaultTarget           string `json:"default_target"` // SAP_IDOC, NETSUITE_CSV, QUICKBOOKS_JSON
	AutoApproveZeroVariance bool   `json:"auto_approve_zero_variance"`
	ExportDirectory         string `json:"export_directory"`
}

// ServerConfig defines HTTP API settings.
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// AppConfig is the root configuration object.
type AppConfig struct {
	Tolerance ToleranceConfig `json:"tolerance"`
	Matching  MatchingConfig  `json:"matching"`
	ERP       ERPConfig       `json:"erp"`
	Server    ServerConfig    `json:"server"`
}

// DefaultConfig returns calibrated enterprise baseline settings.
func DefaultConfig() AppConfig {
	return AppConfig{
		Tolerance: ToleranceConfig{
			MaxPriceVariancePercent:  0.5,
			MaxPriceVarianceAbsolute: 0.05,
			MaxQuantityVariance:      0.0,
			MaxTaxVarianceAbsolute:   0.02,
		},
		Matching: MatchingConfig{
			MinDescriptionSimilarity: 0.60,
			EnableUOMConversion:      true,
			AllowFuzzySKUFallback:    true,
		},
		ERP: ERPConfig{
			DefaultTarget:           "SAP_IDOC",
			AutoApproveZeroVariance: true,
			ExportDirectory:         "./exports",
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 7272,
		},
	}
}

// LoadConfig reads a JSON configuration file, falling back to defaults if not found.
func LoadConfig(path string) (AppConfig, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
