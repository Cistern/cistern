package config

import (
	"encoding/json"
	"io/ioutil"
)

type Configuration struct {
	Sources []ConfigSource `json:"sources"`
}

type ConfigSource struct {
	IP   string            `json:"ip"`
	SNMP *ConfigSourceSNMP `json:"snmp"`
}

type ConfigSourceSNMP struct {
	Port           int    `json:"port"`
	User           string `json:"user"`
	AuthPassphrase string `json:"auth"`
	PrivPassphrase string `json:"priv"`
}

type SNMPEntry struct {
	Address        string `json:"address"`
	User           string `json:"user"`
	AuthPassphrase string `json:"authPassphrase"`
	PrivPassphrase string `json:"privPassphrase"`
}

func Load(path string) (Configuration, error) {
	conf := Configuration{}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return conf, err
	}
	return conf, json.Unmarshal(contents, &conf)
}
