package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type config struct {
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	Database struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Address  string `json:"address"`
		Port     string `json:"port"`
		Name     string `json:"name"`
	} `json:"database"`
	User id `json:"user"`
}

func (cfg *config) getDataSourceName() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", cfg.Database.User,
		cfg.Database.Password, cfg.Database.Address,
		cfg.Database.Port, cfg.Database.Name)
}

func loadConfig(filename string) (*config, error) {
	var cfg config
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
