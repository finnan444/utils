package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadConfiguration универсальный загрузчик конфы
func LoadConfiguration(mode string, config interface{}) error {
	var (
		name     = "conf.json"
		fileName string
		file     *os.File
		err      error
	)

	if mode == "prod" {
		name = "conf_prod.json"
	}

	if fileName, err = filepath.Abs(filepath.Dir("conf/" + name)); err != nil {
		return err
	}

	if file, err = os.Open(fileName); err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(config); err != nil {
		return err
	}
	return nil
}
