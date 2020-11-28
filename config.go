package main

import (
	"fmt"

	"github.com/Necoro/feed2imap-go/pkg/log"
	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml"
)

const configFileName = "bolt-listener.toml"

type dockConfig struct {
	Uuid       string
	Authorize  *cmd
	Disconnect *cmd
}

type config struct {
	Debug bool   `default:"false"`
	Bus   string `default:"system"`
	Docks map[string]dockConfig
}

func loadConfig(configfile string) (cfg *config, err error) {
	if configfile == "" {
		if configfile, err = xdg.ConfigFile(configFileName); err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	}

	log.Debug("Loading config from ", configfile)

	tree, err := toml.LoadFile(configfile)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg = new(config)
	if err = tree.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}
	return
}
