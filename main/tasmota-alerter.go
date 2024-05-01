package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jorycz/tasmota-alerter/pkg/mqttclient"
	"github.com/jorycz/tasmota-alerter/pkg/processor"
)


func main() {
	v, err := ReadEnv()
	if err != nil {
		abort("Error reading env variables, exiting ...", "error", err)
	}

	slog.Info("Connecting to MQTT", "server", v.mqttHost, "port", v.mqttPort, "username", v.mqttUsername)
	mqttClient := mqttclient.NewMqttClient(v.mqttHost, v.mqttPort, v.mqttUsername, v.mqttPassword, v.mqttClientId)
	if err := mqttClient.Connect(); err != nil {
		abort("Error connecting to MQTT broker, exiting ...", "error", err)
	}

	p := processor.NewProcessor(mqttClient, v.statusUpdateSeconds, v.smtpServer)
	if err := p.Subscribe(v.mqttTopics); err != nil {
		abort("Error subscribing topics, exiting ...", "error", err)
	}

	// Create a channel to receive signals - for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	// Notify the channel for specific OS signals
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1)
	go func() {
		for {
			select {
			case sig := <-signalCh:
				osSignalReceived(sig)
			}
		}
	}()

	select {}
}

func osSignalReceived(sig os.Signal) {
	switch sig {
	case os.Interrupt, syscall.SIGTERM:
		// Store already fired alerts to prevent false alarms in case of reboot for example
		processor.StoreFiredAlerts()
		os.Exit(0)
	case syscall.SIGHUP, syscall.SIGUSR1:
		// Dump current alerts to log
		processor.DumpFiredAlerts()
		// Reload rules: kill -HUP $(pidof tasmota-alerter)
		processor.RefreshAlertRules()
	}
}

func abort(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
