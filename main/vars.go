package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type vars struct {
	mqttHost, mqttUsername, mqttPassword, mqttClientId string
	mqttPort, statusUpdateSeconds                      int
	mqttTopics                                         []string
}

func ReadEnv() (*vars, error) {
	v := &vars{}
	v.mqttHost = orDefault(os.Getenv("MQTT_HOSTNAME"), "localhost")
	mqttPort, err := strconv.Atoi(orDefault(os.Getenv("MQTT_PORT"), "1883"))
	if err != nil {
		return nil, fmt.Errorf("can't parse provided mqtt port: %s", err)
	}
	v.mqttPort = mqttPort
	v.mqttUsername = orDefault(os.Getenv("MQTT_USERNAME"), "")
	v.mqttPassword = orDefault(os.Getenv("MQTT_PASSWORD"), "")
	v.mqttClientId = orDefault(os.Getenv("MQTT_CLIENT_ID"), "tasmota_alerter")
	v.mqttTopics = orDefaultList(os.Getenv("MQTT_TOPICS"), "tele/+/+, stat/+/+")
	statusUpdateSeconds, err := strconv.Atoi(orDefault(os.Getenv("STATUS_UPDATE_SECONDS"), "0"))
	if err != nil {
		return nil, fmt.Errorf("can't parse provided status update interval: %s", err)
	}
	v.statusUpdateSeconds = statusUpdateSeconds

	logLevelStr := orDefault(os.Getenv("LOG_LEVEL"), "info")
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
		return nil, fmt.Errorf("can't parse provided log level %q: %s", logLevelStr, err)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))
	return v, nil
}

func orDefault(value string, def string) string {
	if value == "" {
		return def
	}
	return value
}

// read env variable, split per comma, trim the values, and return them as a list of strings
func orDefaultList(value string, def string) []string {
	if value == "" {
		return splitAndTrim(def)
	}
	return splitAndTrim(value)
}

func splitAndTrim(value string) []string {
	s := strings.Split(value, ",")
	for i, v := range s {
		s[i] = strings.TrimSpace(v)
	}
	return s
}
