package email

import (
	"log/slog"
	"net/smtp"
	"strings"
)

func SendMessage(smtpDestination, recpts, body string) {

	recipients := strings.Split(recpts, ",")

	if len(recipients) > 0 {

		from := "tasmota-alerter@localhost"
		subject := "Tasmota Alert"

		for _, to := range recipients {
			slog.Debug("Sending email.", "server", smtpDestination, "to", to, "subj", subject, "message", body)

			to = strings.TrimSpace(to)
			message := []byte("To: " + to + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"\r\n" +
				body + "\r\n")

			err := smtp.SendMail(smtpDestination, nil, from, []string{to}, message)
			if err != nil {
				slog.Error("Error sending e-mail.", err)
			}
		}
	} else {
		slog.Error("Email could not be sent. Recipients list is empty.")
	}
}
