package config

import (
	"encoding/json"
	"io/ioutil"
)

type Configuration struct {
	SNMPDevices []snmpEntry `json:"snmpDevices"`
}

type snmpEntry struct {
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
