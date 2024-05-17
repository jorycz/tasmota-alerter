# tasmota-alerter
Simple alerting daemon designed to work with [Tasmota powered](https://tasmota.github.io/docs/) smart plugs. Inspired (mainly MQTT part) by [tasmota-exporter](https://github.com/dyrkin/tasmota-exporter?tab=readme-ov-file) for Prometheus.
* State of application (already fired alerts) is saved when you need to restart or stop [tasmota-alerter](https://github.com/jorycz/tasmota-alerter), so no alerts should be fired twice. 
* Monitoring rules can be reloaded (if changed) by `kill -HUP $(pidof tasmota-alerter)` if [tasmota-alerter](https://github.com/jorycz/tasmota-alerter) is already running.

# Prerequisites
You need to install [Go](https://go.dev) to compile daemon and [Mosquitto](https://mosquitto.org) where Tasmota devices sending updates using MQTT. Also open SMTP service on **localhost:25** for e-mail notification.

# Installation as a systemd service (Linux)
Default user for installation & run is root. If you wish to change the user, just edit systemd/tasmota-alerter.service after you clone repository.
```
sudo su -
cd /opt/
git clone https://github.com/jorycz/tasmota-alerter.git
cd tasmota-alerter
```
* Edit **systemd/tasmota-alerter.env** and enter mosquitto values like **username** and **password**.
```
MQTT_HOSTNAME:          #optional. Default is localhost.
MQTT_PORT:              #optional. Default is 1883.
MQTT_USERNAME:          #optional. Default is empty.
MQTT_PASSWORD:          #optional. Default is empty.
MQTT_CLIENT_ID:         #optional. Default is tasmota_alerter.
MQTT_TOPICS:            #optional. Default is "tele/+/+, stat/+/+". If you are using deeper topics, you can set as "tele/#, stat/#".
STATUS_UPDATE_SECONDS:  #optional. Default is 0 (disabled). This is how often a status update will be requested.
LOG_LEVEL:              #optional. Default is info. Severity level for log output. Possible values: debug, info, warn, error.
SMTP_SERVER_HOST:       #optional. Default is localhost. Email server hostname / IP.
SMTP_SERVER_PORT:       #optional. Default is 25. Email server port.
```
* Special note about **STATUS_UPDATE_SECONDS**. This function is not needed by default. You can setup plugs to send log every 30s in Logging section of plug GUI. Default is 300s (5 minutes).
```
go build -o tasmota-alerter ./main
cp systemd/tasmota-alerter.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable tasmota-alerter.service
systemctl start tasmota-alerter.service
```

# Setup alerting channels
* E-mail
* Telegram

Check **notifications/** folder. There are example *.conf files. You can make as many files as you wish or just one. Tasmota-alerter reads all files ending with .conf suffix from this folder.

# Setup monitoring rules
Check **rules/** folder. There are example *.conf files. You can make as many files as you wish or just one. Tasmota-alerter reads all files ending with .conf suffix from this folder. One file is prepared for "events", like when someone change state of plug (push ON/OFF button). Another file is prepared for "values" monitoring.
