package main

import (
	"fmt"

	"github.com/Necoro/bolt-listener/pkg/log"
	"github.com/godbus/dbus/v5"
)

// Bolt DBus
const (
	boltSenderName      = "org.freedesktop.bolt"
	boltDeviceInterface = "org.freedesktop.bolt1.Device"
	boltDevicePath      = "/org/freedesktop/bolt/devices/"
	propertiesInterface = "org.freedesktop.DBus.Properties"
)

func (d dockConfig) objectPath() dbus.ObjectPath {
	return dbus.ObjectPath(boltDevicePath + d.Uuid)
}

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

func getStatus(v *dbus.Signal) (string, error) {
	var interfaceName string
	var changed map[string]interface{}
	var invalidated []string

	if err := dbus.Store(v.Body, &interfaceName, &changed, &invalidated); err != nil {
		return "", fmt.Errorf("Unexpected data: %s; Error: %w", v.Body, err)
	}

	if interfaceName != boltDeviceInterface {
		return "", fmt.Errorf("Unexpected Interface: %s", interfaceName)
	}

	if status, ok := changed["Status"]; ok {
		return status.(string), nil
	}

	return "", nil
}
