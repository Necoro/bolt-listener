package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/godbus/dbus/v5"
)

// UUID of the dock as given by `boltctl --list`
const dock = "deadbeef_uuid"

// Scripts to run
const authorizeScript = "connect_dock.sh"
const disconnectScript = "disconnect_dock.sh"

const boltDevice = "org.freedesktop.bolt1.Device"

func run(script string) {
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func authorized() {
	fmt.Println("Authorizing")
	run(authorizeScript)
}

func disconnected() {
	fmt.Println("Disconnecting")
	run(disconnectScript)
}

func main() {
	conn, err := dbus.SystemBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to system bus:", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err = conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/bolt/devices/"+dock),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchSender("org.freedesktop.bolt"),
	); err != nil {
		panic(err)
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for v := range c {
		var interfaceName string
		var changed map[string]interface{}
		var invalidated []string
		if err := dbus.Store(v.Body, &interfaceName, &changed, &invalidated); err != nil {
			panic(fmt.Errorf("Unexpected data: %s; Error: %w", v.Body, err))
		}

		if interfaceName != boltDevice {
			panic(fmt.Errorf("Unexpected Interface: %s", interfaceName))
		}

		if status, ok := changed["Status"]; ok {
			switch status.(string) {
			case "authorized":
				authorized()
			case "disconnected":
				disconnected()
			}
		}
	}
}
