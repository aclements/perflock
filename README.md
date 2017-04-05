Perflock is a simple locking wrapper for running benchmarks on shared
hosts.

To build and install perflock system-wide, run

    $ go get github.com/aclements/perflock/cmd/perflock
    $ sudo install $GOPATH/bin/perflock /usr/bin/perflock
    $ rm $GOPATH/bin/perflock

To start the perflock daemon manually, run

    $ sudo -b perflock -daemon

To enable the perflock daemon on boot, see the instructions for your
init system in the `init/` directory.
