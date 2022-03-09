Perflock is a simple locking wrapper for running benchmarks on shared
hosts.

To build, install, and start perflock system-wide, run

    $ git clone https://github.com/aclements/perflock
    $ cd perflock
    $ go build ./cmd/perflock
    $ sudo ./install.bash

If your init system is supported, this will also configure perflock to
start automatically on boot.

Manual installation
-------------------

To install perflock manually, run

    $ GOBIN=$PWD go install github.com/aclements/perflock/cmd/perflock@latest
    $ sudo install ./perflock /usr/bin/perflock

To start the perflock daemon manually, run

    $ sudo -b perflock -daemon

To enable the perflock daemon on boot, see the instructions for your
init system in the `init/` directory.
