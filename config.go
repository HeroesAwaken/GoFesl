package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Config struct {
	MysqlServer        string
	MysqlUser          string
	MysqlDb            string
	MysqlPw            string
	MysqlLoggingServer string
	MysqlLoggingUser   string
	MysqlLoggingDb     string
	MysqlLoggingPw     string
	InfluxDBHost       string
	InfluxDBDatabase   string
	InfluxDBUser       string
	InfluxDBPassword   string
}

func (config *Config) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	return nil
}

func (config *Config) Load(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
}
