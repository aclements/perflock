To configure Upstart to run perflock, copy `perflock.conf` into
`/etc/init/` and run `sudo start perflock`. E.g.,

    $ sudo install -m 0644 perflock.conf /etc/init/
    $ sudo start perflock
