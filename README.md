# bolt-listener

`bolt-listener` is a daemon to be used in combination with [bolt](https://gitlab.freedesktop.org/bolt/bolt). It uses DBus to listen whenever bolt announces the connection / disconnection of some thunderbolt hardware.
It then executes the scripts configured.

A common usecase is probably to do some monitor setup or alike when connecting with a Thunderbolt dock (and resetting it, when disconnecting).

Configuration happens via [TOML](https://toml.io), an [example config](example.toml) is included.
