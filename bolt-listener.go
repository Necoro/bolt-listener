package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/Necoro/feed2imap-go/pkg/log"
	"github.com/adrg/xdg"
	"github.com/godbus/dbus/v5"
	"github.com/pelletier/go-toml"
)

const configFileName = "bolt-listener.toml"

// Bolt DBus
const (
	boltSenderName      = "org.freedesktop.bolt"
	boltDeviceInterface = "org.freedesktop.bolt1.Device"
	boltDevicePath      = "/org/freedesktop/bolt/devices/"
	propertiesInterface = "org.freedesktop.DBus.Properties"
)

type dockConfig struct {
	Uuid       string
	Authorize  string
	Disconnect string
}

func (d dockConfig) objectPath() dbus.ObjectPath {
	return dbus.ObjectPath(boltDevicePath + d.Uuid)
}

type config struct {
	Debug bool   `default:"false"`
	Bus   string `default:"system"`
	Docks map[string]dockConfig
}

type dock struct {
	uuid       string
	authorize  string
	disconnect string
	name       string
}

type docks map[dbus.ObjectPath]dock

func execScript(script string) error {
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return cmd.Run()
}
func (d *dock) authorized() error {
	log.Debug("Authorizing ", d.name)
	if err := execScript(d.authorize); err != nil {
		return fmt.Errorf("authorizing %s: %w", d.name, err)
	}
	return nil
}

func (d *dock) disconnected() error {
	log.Debug("Disconnecting ", d.name)
	if err := execScript(d.disconnect); err != nil {
		return fmt.Errorf("disconnecting %s: %w", d.name, err)
	}
	return nil
}

func (d *dock) handleStatus(status string) error {
	switch status {
	case "authorized":
		return d.authorized()
	case "disconnected":
		return d.disconnected()
	default:
		return nil
	}
}

func loadConfig(configfile string) (cfg *config, err error) {
	if configfile == "" {
		if configfile, err = xdg.ConfigFile(configFileName); err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	}

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

func dbusConnect(busName string) (*dbus.Conn, error) {
	if busName == "session" {
		return dbus.SessionBus()
	} else {
		return dbus.SystemBus()
	}
}

func addSignal(conn *dbus.Conn, path dbus.ObjectPath) error {
	return conn.AddMatchSignal(
		dbus.WithMatchObjectPath(path),
		dbus.WithMatchInterface(propertiesInterface),
		dbus.WithMatchSender(boltSenderName),
	)
}

func run() error {
	cfgOverride := flag.String("f", "", "configuration file")
	debug := flag.Bool("d", false, "enable debug output")
	flag.Parse()

	config, err := loadConfig(*cfgOverride)
	if err != nil {
		return err
	}

	if config.Debug || *debug {
		log.SetDebug()
	} else {
		log.SetVerbose()
	}

	conn, err := dbusConnect(config.Bus)
	if err != nil {
		return fmt.Errorf("failed to connect to %s bus: %w", config.Bus, err)
	}
	defer conn.Close()

	docks := make(docks)
	for name, d := range config.Docks {
		docks[d.objectPath()] = dock{
			uuid:       d.Uuid,
			authorize:  d.Authorize,
			disconnect: d.Disconnect,
			name:       name,
		}
		if err = addSignal(conn, d.objectPath()); err != nil {
			return fmt.Errorf("failed to listen to signal for %s: %w", name, err)
		}
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for v := range c {
		dock, ok := docks[v.Path]
		if !ok {
			log.Print("Ignoring unexpected path ", v.Path)
			continue
		}

		var interfaceName string
		var changed map[string]interface{}
		var invalidated []string
		if err = dbus.Store(v.Body, &interfaceName, &changed, &invalidated); err != nil {
			return fmt.Errorf("Unexpected data: %s; Error: %w", v.Body, err)
		}

		if interfaceName != boltDeviceInterface {
			return fmt.Errorf("Unexpected Interface: %s", interfaceName)
		}

		if status, ok := changed["Status"]; ok {
			if err = dock.handleStatus(status.(string)); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
