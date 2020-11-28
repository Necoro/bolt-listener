package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Necoro/feed2imap-go/pkg/log"
	"github.com/godbus/dbus/v5"
)

type cmd struct {
	Exe  string `toml:"cmd"`
	Args []string
}

func (c *cmd) exec() error {
	log.Debugf("Executing '%s'", c.Exe)

	cmd := exec.Command(c.Exe, c.Args...)
	if log.IsDebug() {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type dock struct {
	uuid       string
	authorize  *cmd
	disconnect *cmd
	name       string
}

type docks map[dbus.ObjectPath]dock

func (d *dock) handle(script *cmd, descr string) error {
	if script == nil {
		log.Debugf("Ignoring %s for %s", descr, d.name)
		return nil
	}
	log.Debugf("%s for %s", strings.Title(descr), d.name)

	err := script.exec()
	if err != nil {
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
		if d.Uuid == "" {
			return fmt.Errorf("UUID is mandatory, but missing for %s.", name)
		}

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

		status, err := getStatus(v)
		if err != nil && status != "" {
			err = dock.handleStatus(status)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

var testMode = false

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-t" {
		testMode = true
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	if err := run(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
