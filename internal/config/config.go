/*
 * Copyright (C) 2026 Russ Shingleton <reshingleton@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Secrets      []string       `yaml:"secrets"`
	Listen       string         `yaml:"listen,omitempty"`
	SQLiteDSN    string         `yaml:"sqlite_dsn,omitempty"`
	SessionStore string         `yaml:"session_store,omitempty"`
	Production   bool           `yaml:"production,omitempty"`
	Database     DatabaseConfig `yaml:"database,omitempty"`
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Type    string `yaml:"type"`
	Path    string `yaml:"path"`
	WALMode bool   `yaml:"wal_mode,omitempty"`
}

var appConfig *Config

// Load reads configuration from the YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Listen == "" {
		cfg.Listen = ":3009"
	}
	if cfg.SQLiteDSN == "" {
		cfg.SQLiteDSN = "data/rescms.db"
	}

	appConfig = &cfg
	return appConfig, nil
}

// Get returns the loaded configuration
func Get() *Config {
	return appConfig
}
