package config

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
)


func ConfInit() (*ini.File, error) {
	ConfigFile := "./config/VCmonitor.ini"
	cfg, err := ini.Load(ConfigFile)
	if err != nil {
		fmt.Println("config file open fail")
		os.Exit(500)
	}
	return cfg,err
}

