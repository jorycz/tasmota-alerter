package ruleengine

import (
	"log/slog"
	"strings"
	"sync"
)

var (
	monitoringRules map[string][]Rule
	rulesProcessed  func() int
	lock            sync.Mutex
)

type Rule struct {
	JsonPathOrEventTag  string
	CompareValue        string
	Recipients          string
	MessageRuleActive   string
	MessageRuleInActive string
}

type Rules struct {
	MonitoringRules map[string][]Rule
}

func NewRules() *Rules {

	monitoringRules = make(map[string][]Rule)
	readRuleFiles()

	return &Rules{monitoringRules}
}

func RefreshRules() {
	readRuleFiles()
}


func readRuleFiles() {
	ruleFilesLines, err := readRuleConfFiles()
	if err != nil {
		slog.Error("Error when reading RULE FILES", err)
	}
	createUniversalRuleSet(ruleFilesLines)
}

func createUniversalRuleSet(ruleLines []string) {
	lock.Lock()
	for k := range monitoringRules {
		delete(monitoringRules, k)
	}
	lock.Unlock()

	rulesProcessed = incrementSeqNumber()

	for _, line := range ruleLines {
		slog.Debug("Loading monitoring rule.", "data", line)
		parsed := strings.Split(line, ":::")
		if len(parsed) > 2 {

			device := strings.Split(line, ":::")[0]

			r := Rule{}
			r.JsonPathOrEventTag = strings.Split(line, ":::")[1]
			r.CompareValue = strings.Split(line, ":::")[2]
			if len(parsed) > 3 {
				r.Recipients = strings.Split(line, ":::")[3]
			}
			if len(parsed) > 4 {
				r.MessageRuleActive = strings.Split(line, ":::")[4]
			}
			if len(parsed) > 5 {
				r.MessageRuleInActive = strings.Split(line, ":::")[5]
			}
			_ = rulesProcessed()
			monitoringRules[device] = append(monitoringRules[device], r)
		} else {
			slog.Error("Can not parse rule!", "rule_line", line)
		}
	}

	slog.Info("Rules loaded.", "count", rulesProcessed())
}

func incrementSeqNumber() func() int {
	num := -1
	return func() int {
		num++
		return num
	}
}
