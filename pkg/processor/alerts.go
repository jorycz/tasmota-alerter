package processor

import (
	"encoding/json"
	"log/slog"
	"os"
)

const firedAlertLastStateStorage = "storage/firedAlerts.json"

type Alert struct {
	AlertJsonPathOrEventTag      string
	AlertMonitoredActionAndValue string
	Recipients                   string
}

type Alerts struct {
	FiredAlerts map[string][]Alert
}

func NewAlerts() Alerts {

	alerts := readAlerts()
	if len(alerts.FiredAlerts) < 1 {
		alerts.FiredAlerts = make(map[string][]Alert)
	} else {
		slog.Info("Previously fired alerts restored.", "file", firedAlertLastStateStorage)
	}

	slog.Debug("NewAlerts", "alerts", alerts)
	return Alerts{alerts.FiredAlerts}
}

func (alerts Alerts) StoreAlerts() {
	slog.Info("Storing already fired alerts.", "file", firedAlertLastStateStorage)

	jsonData, err := json.Marshal(alerts)
	if err != nil {
		slog.Error("Error encoding JSON when storing fired alerts:", err)
		return
	}

	file, err := os.Create(firedAlertLastStateStorage)
	if err != nil {
		slog.Error("Error creating file when storing fired alerts:", err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		slog.Error("Error writing JSON to file when storing fired alerts:", err)
		return
	}
}

func readAlerts() Alerts {

	if _, err := os.Stat(firedAlertLastStateStorage); os.IsNotExist(err) {
		slog.Debug("File with stored alerts does not exist or is not readable.")
		return Alerts{}
	}

	file, err := os.Open(firedAlertLastStateStorage)
	if err != nil {
		slog.Error("Error opening file with stored alerts:", err)
		return Alerts{}
	}
	defer file.Close()

	var restoredData Alerts
	err = json.NewDecoder(file).Decode(&restoredData)
	if err != nil {
		slog.Error("Error decoding JSON from stored alerts:", err)
		return Alerts{}
	}

	return restoredData
}
