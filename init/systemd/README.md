To configure systemd to run perflock, run

    $ sudo install -m 0644 perflock.service /etc/systemd/system
    $ sudo systemctl enable --now perflock.service
