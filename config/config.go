package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// продакшн режим, в докер как env variable пробрасываем
const ProdMode = "prod"

// LoadConfiguration универсальный загрузчик конфы
func LoadConfiguration(mode string, config interface{}) (err error) {
	var (
		name    = "conf.json"
		fileDir string
		file    *os.File
	)

	if mode == ProdMode {
		name = "conf_prod.json"
	}

	if fileDir, err = filepath.Abs(filepath.Dir(fmt.Sprintf("conf/%s", name))); err != nil {
		return
	}

	if file, err = os.Open(fmt.Sprintf("%s/%s", fileDir, name)); err != nil {
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(config); err != nil {
		return
	}

	return
}
