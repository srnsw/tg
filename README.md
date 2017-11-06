tg serves up teamgage dashboards as simple png files. It comprises two services: tgserve (a simple webserver that serves the dashboard images) and tgupdate (uses chrome headless browser to scrape teamgage).

# Install

Install golang (https://golang.org/doc/install).

Install Google Chrome (https://www.google.com/chrome/browser/desktop/index.html). After finding the most recent deb file, do:

    sudo dpkg -i google-chrome-stable_current_amd64.deb
    sudo apt-get -f install

Install tg binaries:

    go get github.com/srnsw/tg/tgupdate
    go get github.com/srnsw/tg/tgserve

Copy tg binaries to /usr/local/bin/ then:

    sudo chown root /usr/local/bin/tgserve
    sudo chown root /usr/local/bin/tgupdate

Install these systemd files by copying to /etc/systemd/system/ folder:

    chrome.service
    teamgage-serve.service
    teamgage-update.service
    teamgage-update.timer

Enable teamgage-serve and teamgage-update.timer:

    sudo systemctl enable teamgage-serve.service
    sudo systemctl enable teamgage-update.timer

Start the services with systemctl start or just reboot.

# Systemctl cheats

  - To start: sudo systemctl start application.service | application.timer
  - To restart: sudo systemctl restart application.service
  - To enable loading at boot: sudo systemctl enable application.service
  - To disable: sudo systemctl disable application.service
  - Check status: systemctl status application.service

  For more, I like this [Digital Ocean tutorial](https://www.digitalocean.com/community/tutorials/understanding-systemd-units-and-unit-files).