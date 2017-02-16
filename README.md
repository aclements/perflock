Perflock is a simple locking wrapper for running benchmarks on shared
hosts.

To build and install perflock system-wide, run

    $ go get github.com/aclements/perflock/cmd/perflock
    $ sudo mv $GOPATH/bin/perflock /usr/bin/perflock
    $ sudo chmod a+x /usr/bin/perflock

To enable the perflock daemon on boot, see the instructions for your
init system in the `init/` directory.
