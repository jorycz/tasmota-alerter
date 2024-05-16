package notificationengine

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/jorycz/tasmota-alerter/pkg/utils"
)

var (
	notificationChannels map[string][]string
	rulesProcessed       func() int
	lock                 sync.Mutex
	smtpSendingServer    string
)

func SetupChannels(smtp string) {
	smtpSendingServer = smtp
	notificationChannels = make(map[string][]string)
	readConfigFiles()
}

func NotifyChannels(channels string, message string) {
	notifyChannels := strings.Split(channels, ",")
	for _, channel := range notifyChannels {

		if strings.HasPrefix(channel, "EMAIL") {
			sendEmailWithMessage(notificationChannels[channel], message)
		}
		if strings.HasPrefix(channel, "TELEGRAM") {
			sendTelegramWithMessage(notificationChannels[channel], message)
		}
	}
}

func readConfigFiles() {
	ruleFilesLines, err := utils.ReadFilesWithSuffix("notifications", ".conf")
	if err != nil {
		slog.Error("Error when reading RULE FILES", err)
	}
	createUniversalRuleSet(ruleFilesLines)
}

func createUniversalRuleSet(ruleLines []string) {
	lock.Lock()
	for k := range notificationChannels {
		delete(notificationChannels, k)
	}
	lock.Unlock()

	rulesProcessed = incrementSeqNumber()

	for _, line := range ruleLines {
		slog.Debug("Loading notification rule.", "data", line)
		parsed := strings.Split(line, ":::")
		if len(parsed) > 1 {
			notificationChannels[parsed[0]] = parsed[1:]
			slog.Debug("CHANNEL", "line", parsed)
			_ = rulesProcessed()
		} else {
			slog.Error("Can not parse notification!", "notification_line", line)
		}
	}

	slog.Info("Notification channles loaded.", "count", rulesProcessed())
}

func sendEmailWithMessage(recipients []string, message string) {
	sendEmailMessage(smtpSendingServer, recipients, message)
}

func sendTelegramWithMessage(botTokenAndChatId []string, message string) {
	sendTelegramMessage(botTokenAndChatId[0], botTokenAndChatId[1], message)
}

func incrementSeqNumber() func() int {
	num := -1
	return func() int {
		num++
		return num
	}
}
