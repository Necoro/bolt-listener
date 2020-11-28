#!/bin/sh

status=$1

busctl emit \
    --user \
    /org/freedesktop/bolt/devices/123456 \
    org.freedesktop.DBus.Properties \
        PropertiesChanged \
        'sa{sv}as' \
        org.freedesktop.bolt1.Device \
        1 \
            Status s $status \
        0
