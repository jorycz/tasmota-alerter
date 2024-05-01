package processor

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/jorycz/sp-json"
	"github.com/jorycz/tasmota-alerter/pkg/email"
	"github.com/jorycz/tasmota-alerter/pkg/mqttclient"
	"github.com/jorycz/tasmota-alerter/pkg/ruleengine"
)

const ruleNotificationSytemTag = "__SYSTEM__"

type Processor struct {
	scheduled           map[string]any
	lock                *sync.Mutex
	mqttClient          *mqttclient.MqttClient
	statusUpdateSeconds int
	jsonParser          *parser.JSONParser
	ruleEngineRules     *ruleengine.Rules
}

var (
	firedAlertStorage Alerts
	smtpDestination   string
)

func NewProcessor(mqttClient *mqttclient.MqttClient, statusUpdateSeconds int, smtpServer string) *Processor {
	firedAlertStorage = NewAlerts()
	smtpDestination = smtpServer
	return &Processor{map[string]any{}, &sync.Mutex{}, mqttClient, statusUpdateSeconds, &parser.JSONParser{}, ruleengine.NewRules()}
}

func RefreshAlertRules() {
	ruleengine.RefreshRules()
}

func StoreFiredAlerts() {
	firedAlertStorage.StoreAlerts()
}

func DumpFiredAlerts() {
	firedAlertStorage.DumpAlerts()
}

func (p *Processor) Subscribe(mqttListenTopics []string) error {
	for _, topic := range mqttListenTopics {
		if err := p.mqttClient.Subscribe(topic, p.messageProcessor); err != nil {
			return fmt.Errorf("can't subscribe to %q: %w", topic, err)
		} else {
			slog.Info("Topic subscribed.", "topic", topic)
		}
	}
	return nil
}

func (p *Processor) messageProcessor(_ mqtt.Client, m mqtt.Message) {
	slog.Debug("MQTT Message arrived", "topic", m.Topic(), "payload", m.Payload())

	// In case /STATUS0 with additional information is needed periodically from device - default OFF in vars.go
	if p.statusUpdateSeconds > 0 {
		p.scheduleStatusCommand(m.Topic())
	}

	// Process message topic & payload but ignore Keep-Alive messages
	if !strings.HasSuffix(m.Topic(), "/LWT") {

		topicParts := strings.Split(m.Topic(), "/")
		if len(topicParts) > 2 {
			// Topic is 3-parts like: tele/plug_washing-machine/SENSOR
			deviceTopic := topicParts[1]
			monitoringRulesForDevice := p.ruleEngineRules.MonitoringRules[deviceTopic]
			if len(monitoringRulesForDevice) > 0 {
				// Any monitoring rules found for this device
				p.compareRulesWithPayload(topicParts, monitoringRulesForDevice, m.Payload())
			}
		}
	}
}

func (p *Processor) compareRulesWithPayload(topicParts []string, rulesForDevice []ruleengine.Rule, messagePayload []byte) {

	deviceTopic := topicParts[1]
	// Suffixes could be /STATE, /SENSOR (periodicaly reported), /STATUS0 (on demand - check statusUpdateSeconds) or suffix for events like /POWER
	deviceSuffix := fmt.Sprintf("/%v", topicParts[2])

	for _, rule := range rulesForDevice {

		// EVENT-BASED - suffix monitoring like .../POWER events
		if deviceSuffix == rule.CompareValue {
			notifyMonitoredEventArrived(rule.Recipients, fmt.Sprintf("%v %v", rule.MessageRuleActive, string(messagePayload[:])))
			continue
		}

		// LOG-BASED - monitorong based on values of JSON key
		// Get current value of JSON key on JSON path from device payload
		deviceValue, err := p.jsonParser.GetValueOfJsonKeyOnPath(messagePayload, jsonPathAsArrayElements(rule.JsonPathOrEventTag))
		if err != nil {
			slog.Error("Error when processing JSON Path", err, "device", deviceTopic, "json_path", rule.JsonPathOrEventTag, "json_payload", messagePayload)
			continue
		}

		// ------ If JSON value is found, let's compare it with current rule ------
		if deviceValue != nil {
			slog.Debug("DEBUG - Comparing device data with rule", "topic", deviceTopic, "suffix", deviceSuffix, "json_path", rule.JsonPathOrEventTag, "deviceValue", deviceValue, "rule_value", rule.CompareValue)

			// Split monitoring value to comparison and value itself
			if len(rule.CompareValue) > 1 {
				ruleComparison := rule.CompareValue[:1]
				ruleValue := rule.CompareValue[1:]
				// Check if value in rule can be parsed to number. If not, just compare strings.
				//   THEN if the conditions are met - notify (if not notified before) & store rule details to alert storage
				//   OR if the conditions are NOT met - try to remove alert from stored alerts
				ruleValueIsNumer, err := strconv.ParseFloat(ruleValue, 64)
				if err != nil {
					// Rule values is STRING
					if ruleValue == deviceValue {
						notifyMonitoredValueArrived(deviceTopic, deviceValue.(string), rule)
					} else {
						removeAlertIfNotifiedBefore(deviceTopic, deviceValue.(string), rule)
					}
				} else {
					// Rule values is NUMBER
					if ruleComparison == "=" {
						if deviceValue.(float64) == ruleValueIsNumer {
							notifyMonitoredValueArrived(deviceTopic, fmt.Sprintf("%.3f", deviceValue), rule)
						} else {
							removeAlertIfNotifiedBefore(deviceTopic, fmt.Sprintf("%.3f", deviceValue), rule)
						}
					}
					if ruleComparison == ">" {
						if deviceValue.(float64) > ruleValueIsNumer {
							notifyMonitoredValueArrived(deviceTopic, fmt.Sprintf("%.3f", deviceValue), rule)
						} else {
							removeAlertIfNotifiedBefore(deviceTopic, fmt.Sprintf("%.3f", deviceValue), rule)
						}
					}
					if ruleComparison == "<" {
						if deviceValue.(float64) < ruleValueIsNumer {
							notifyMonitoredValueArrived(deviceTopic, fmt.Sprintf("%.3f", deviceValue), rule)
						} else {
							removeAlertIfNotifiedBefore(deviceTopic, fmt.Sprintf("%.3f", deviceValue), rule)
						}
					}
				}
			}
		}
	}
}

func jsonPathAsArrayElements(jsonPath string) []string {
	return strings.Split(jsonPath, "-->")
}

func lastJsonPathComponentKeyName(jsonPath string) string {
	lastIndex := strings.LastIndex(jsonPath, "-->")
	if lastIndex != -1 {
		return jsonPath[lastIndex+len("-->"):]
	}
	return ""
}

func notifyMonitoredEventArrived(recipients string, emailBody string) {
	if len(recipients) > 0 {
		email.SendMessage(smtpDestination, recipients, emailBody)
	}
}

func notifyMonitoredValueArrived(device string, deviceValue string, rule ruleengine.Rule) {
	if len(rule.Recipients) > 0 && !isRuleForThisDeviceAlreadyAlerted(device, rule) {
		monitoredValueKeyName := lastJsonPathComponentKeyName(rule.JsonPathOrEventTag)
		// Default email system message (or if no field is specified in rule file)
		emailBody := fmt.Sprintf("Status of [ %v ] changed. Detected value for key [ %v ] is [ %v ] and monitored condition is [ %v ].", device, monitoredValueKeyName, deviceValue, rule.CompareValue)
		if len(rule.MessageRuleActive) > 0 && rule.MessageRuleActive != ruleNotificationSytemTag {
			emailBody = rule.MessageRuleActive
		}
		email.SendMessage(smtpDestination, rule.Recipients, emailBody)
	}
}

func isRuleForThisDeviceAlreadyAlerted(device string, rule ruleengine.Rule) bool {

	storedAlerts := firedAlertStorage.FiredAlerts[device]
	for idx, alert := range storedAlerts {
		if len(alert.AlertJsonPathOrEventTag) > 0 && len(alert.AlertMonitoredActionAndValue) > 0 {
			if alert.AlertJsonPathOrEventTag == rule.JsonPathOrEventTag && alert.AlertMonitoredActionAndValue == rule.CompareValue && rule.Recipients == alert.Recipients {

				if alert.IgnoreCount > 0 {
					alert.IgnoreCount -= 1
					// Update alert
					firedAlertStorage.FiredAlerts[device] = arrayWithDeletedElementAtIndex(storedAlerts, idx)
					firedAlertStorage.FiredAlerts[device] = append(firedAlertStorage.FiredAlerts[device], alert)
					if alert.IgnoreCount == 0 {
						slog.Debug("ALERT - Ignore count reached ZERO, alerting ...", "device", device, "alert", alert)
						return false
					}
					// Ignore count is NOT ZERO yet
					slog.Debug("ALERT - Ignore count NOT ZERO yet, ignoring ...", "device", device, "alert", alert)
					return true
				} else {
					// Notified already
					slog.Debug("ALERT - Already notified, ignoring ...", "device", device, "alert", alert)
					return true
				}
			}
		}
	}

	newAlert := Alert{}
	newAlert.IgnoreCount = rule.IgnoreOccurrences
	newAlert.AlertJsonPathOrEventTag = rule.JsonPathOrEventTag
	newAlert.AlertMonitoredActionAndValue = rule.CompareValue
	newAlert.Recipients = rule.Recipients
	firedAlertStorage.FiredAlerts[device] = append(firedAlertStorage.FiredAlerts[device], newAlert)

	if newAlert.IgnoreCount == 0 {
		slog.Debug("ALERT - NEW. Rule ignore count is ZERO, alerting ...", "device", device, "alert", newAlert)
		return false
	} else {
		// Ignore alerting now, fake that it has been alerted and take care about counter & alerting next time
		slog.Debug("ALERT - NEW. Ignore counter active, ignoring ...", "device", device, "alert", newAlert)
		return true
	}
}

func removeAlertIfNotifiedBefore(device string, deviceValue string, rule ruleengine.Rule) {
	storedAlerts := firedAlertStorage.FiredAlerts[device]
	for idx, alert := range storedAlerts {
		if len(alert.AlertJsonPathOrEventTag) > 0 && len(alert.AlertMonitoredActionAndValue) > 0 {
			if alert.AlertJsonPathOrEventTag == rule.JsonPathOrEventTag && alert.AlertMonitoredActionAndValue == rule.CompareValue && alert.Recipients == rule.Recipients {
				firedAlertStorage.FiredAlerts[device] = arrayWithDeletedElementAtIndex(storedAlerts, idx)
				slog.Debug("ALERT - Removed.", "device", device, "alert", alert)

				if len(rule.Recipients) > 0 {
					// Send notification when returned to normal state only when field is specified in rule file
					if len(rule.MessageRuleInActive) > 0 {
						monitoredValueKeyName := lastJsonPathComponentKeyName(rule.JsonPathOrEventTag)
						// Default email system message
						emailBody := fmt.Sprintf("Status of [ %v ] changed. Detected value for key [ %v ] is [ %v ] and monitored condition is [ %v ].", device, monitoredValueKeyName, deviceValue, rule.CompareValue)
						if rule.MessageRuleInActive != ruleNotificationSytemTag {
							emailBody = rule.MessageRuleInActive
						}
						email.SendMessage(smtpDestination, rule.Recipients, emailBody)
					}
				}
			}
		}
	}
}

func arrayWithDeletedElementAtIndex(arr []Alert, index int) []Alert {
	return append(arr[:index], arr[index+1:]...)
}

func (p *Processor) scheduleStatusCommand(topic string) {
	if strings.HasPrefix(topic, "tele/") && strings.HasSuffix(topic, "/STATE") {
		segments := strings.Split(topic, "/")
		source := strings.Join(segments[1:len(segments)-1], "/")
		segments[0], segments[len(segments)-1] = "cmnd", "Status0"
		target := strings.Join(segments, "/")
		p.lock.Lock()
		defer p.lock.Unlock()
		if _, ok := p.scheduled[source]; !ok {
			slog.Debug("Scheduling status updates", "seconds", p.statusUpdateSeconds, "source", source)
			p.scheduled[source] = true
			ticker := time.NewTicker(time.Duration(p.statusUpdateSeconds) * time.Second)
			go func() {
				for {
					select {
					case <-ticker.C:
						slog.Debug("Sending status update request", "command", target)
						if err := p.mqttClient.SendCommand(target, ""); err != nil {
							slog.Error("Can't send status command", "command", target)
						}
					}
				}
			}()
		}
	}
}
