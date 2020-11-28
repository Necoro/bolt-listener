package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	log.Debugf("Executing '%s'", script)

	cmd := exec.Command(script)
	if log.IsDebug() {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *dock) handle(script, descr string) error {
	if script == "" {
		log.Debugf("Ignoring %s for %s", descr, d.name)
		return nil
	}

	log.Debugf("%s for %s", strings.Title(descr), d.name)
	if err := execScript(script); err != nil {
		return fmt.Errorf("%s for %s: %w", descr, d.name, err)
	}
	return nil
}

func (d *dock) handleStatus(status string) error {
	switch status {
	case "authorized":
		return d.handle(d.authorize, "authorize")
	case "disconnected":
		return d.handle(d.disconnect, "disconnect")
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

func dbusConnect(busName string) (*dbus.Conn, error) {
	log.Debugf("Connecting to the %s bus", busName)

	switch busName {
	case "session":
		return dbus.SessionBus()
	case "system":
		return dbus.SystemBus()
	default:
		return nil, fmt.Errorf("unknown bus '%s'", busName)
	}
}

var testMode = false

func addSignal(conn *dbus.Conn, path dbus.ObjectPath) error {
	log.Debug("Watching for signals of ", path)

	options := []dbus.MatchOption{
		dbus.WithMatchObjectPath(path),
		dbus.WithMatchInterface(propertiesInterface),
	}

	if !testMode {
		options = append(options, dbus.WithMatchSender(boltSenderName))
	}

	return conn.AddMatchSignal(options...)
}

func run() error {
	cfgOverride := flag.String("f", "", "configuration file")
	debug := flag.Bool("d", false, "enable debug output")
	flag.Parse()

	if *debug { // need this for loading the config
		log.SetDebug()
	}

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
		log.Debug("Received: ", v)

		dock, ok := docks[v.Path]
		if !ok {
			return fmt.Errorf("unexpected path %s", v.Path)
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
	if len(os.Args) > 2 && os.Args[1] == "-t" {
		testMode = true
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	if err := run(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}