tg serves up teamgage dashboards as simple png files. It comprises two services: tgserve (a simple webserver that serves the dashboard images) and tgupdate (uses chrome headless browser to scrape teamgage).

# Install

Install golang

Install Google Chrome (https://www.google.com/chrome/browser/desktop/index.html):

    sudo dpkg -i google-chrome-stable_current_amd64.deb
    sudo apt-get -f install

copy tg binaries to /usr/local/bin/

    sudo chown root /usr/local/bin/tgserve
    sudo chmod +s /usr/local/bin/tgserve
    sudo chown root /usr/local/bin/tgupdate
    sudo chmod +s /usr/local/bin/tgupdate

(maybe `sudo setcap cap_net_raw=eip /usr/local/bin/tgserve`)

Install these systemd services files:

  - /etc/systemd/system/chrome.service
  - /etc/systemd/system/teamgage-serve.service
  - /etc/systemd/system/teamgage-update.service

Start and enable teamgage-serve and teamgage-update.timer.

Systemctl cheats:

  - To start: sudo systemctl start application.service | application.timer
  - To restart: sudo systemctl restart application.service
  - To enable loading at boot: sudo systemctl enable application.service
  - To disable: sudo systemctl disable application.service
  - Check status: systemctl status application.service

  For more, I like this [Digital Ocean tutorial](https://www.digitalocean.com/community/tutorials/understanding-systemd-units-and-unit-files).