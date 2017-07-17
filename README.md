tg serves up teamgage dashboards as simple png files

# Install

Install golang and Chrome (https://www.google.com/chrome/browser/desktop/index.html \ sudo dpkg -i google-chrome-stable_current_amd64.deb \ sudo apt-get -f install)

copy tg binary to /usr/local/bin/tg

`sudo chown root /usr/local/bin/tg`
`sudo chmod +s /usr/local/bin/tg`

(maybe `sudo setcap cap_net_raw=eip /usr/local/bin/tg`)

two services files:

/etc/systemd/system/chrome.service
[Unit]
Description=Chrome headless service
Before=teamgage.service

[Service]
User=root
ExecStart=/usr/bin/google-chrome --headless --disable-gpu --remote-debugging-port=9222
Restart=always

[Install]
WantedBy=teamgage.service

/etc/systemd/system/teamgage.service
[Unit]
Description=Chrome headless service
Before=teamgage.service

[Service]
User=root
ExecStart=/usr/bin/google-chrome --headless --disable-gpu --remote-debugging-port=9222
Restart=always

[Install]
WantedBy=teamgage.service
richard_lehane@instance-3:/etc/systemd/system$ cat teamgage.service
[Unit]
Description=Teamgage screenshoting service
Wants=chrome.service

[Service]
User=root
Environment="TGTEAM=XXXXX"
Environment="TGUSER=XXXXX"
Environment="TGPASS=XXXXX"
Environment="TGPATH=/var/lib/teamgage"
ExecStart=/usr/local/bin/tg
Restart=always

[Install]
WantedBy=multi-user.target

To restart: sudo systemctl restart application.service
To enable loading at boot: sudo systemctl enable application.service