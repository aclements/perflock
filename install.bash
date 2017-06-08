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

if [[ -d /etc/init ]]; then
    echo "Installing init script for Upstart" 1>&2
    sudo install -m 0644 init/upstart/perflock.conf /etc/init/
fi

if /usr/bin/perflock -list 1>&2 >/dev/null; then
    echo "Not starting perflock daemon (already running)" 1>&2
else
    echo "Starting perflock daemon" 1>&2
    sudo -b /usr/bin/perflock -daemon
fi
