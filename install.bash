#!/bin/bash

set -e

BINPATH="$(go env GOBIN)"
if [[ -z "$BINPATH" ]]; then
    BINPATH="$(go env GOPATH)/bin"
fi
BIN="$BINPATH/perflock"
if [[ ! -x "$BIN" ]]; then
    echo "perflock binary $BIN does not exist." 2>&1
    echo "Please run go install github.com/aclements/perflock/cmd/perflock" 2>&1
    exit 1
fi

echo "Installing $BIN to /usr/bin" 1>&2
sudo install "$BIN" /usr/bin/perflock

start="-b /usr/bin/perflock -daemon"
starttype=
if [[ -d /etc/init ]]; then
    echo "Installing init script for Upstart" 1>&2
    sudo install -m 0644 init/upstart/perflock.conf /etc/init/
    start="service perflock start"
    starttype=" (using Upstart)"
fi
if [[ -d /etc/systemd ]]; then
    echo "Installing service for systemd" 1>&2
    sudo install -m 0644 init/systemd/perflock.service /etc/systemd/system
    sudo systemctl enable --quiet perflock.service
    start="systemctl start perflock.service"
    starttype=" (using systemd)"
fi

if /usr/bin/perflock -list >/dev/null 2>&1; then
    echo "Not starting perflock daemon (already running)" 1>&2
else
    echo "Starting perflock daemon$starttype" 1>&2
    sudo $start
fi
