package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type dbConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Address  string `json:"address"`
	Port     string `json:"port"`
	Name     string `json:"name"`
}

func (cfg *dbConfig) getDataSourceName() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", cfg.User, cfg.Password, cfg.Address, cfg.Port, cfg.Name)
}

func loadConfig(filename string) (*dbConfig, error) {
	var cfg dbConfig
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
