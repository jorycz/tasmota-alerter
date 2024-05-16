package notificationengine

import (
	"fmt"
	"log/slog"
	"encoding/json"

	"github.com/jorycz/tasmota-alerter/pkg/http"
)

// JSON struct for response
type JsonResponse struct {
	OK bool `json:"ok"`
}

func sendTelegramMessage(botToken, chatId, message string) {
	slog.Debug("TELEGRAM", "botToken", botToken, "chatId", chatId, "message", message)

	jsonData := fmt.Sprintf(`{"chat_id": "%v", "text": "%v"}`, chatId, message)
	dstUrl := fmt.Sprintf(`https://api.telegram.org/%v/sendMessage`, botToken)

	http.SetHeader("Content-Type: application/json")

	httpStatusCode, responseBody := http.CallUrlWithOptionalTextData(dstUrl, jsonData)

	var httpBodyFinal string

	if httpStatusCode > 0 && httpStatusCode < 400 {
		// Read the response in JSON format
		var tokenResponse JsonResponse
		decoder := json.NewDecoder(responseBody)
		err := decoder.Decode(&tokenResponse)
		if err != nil {
			slog.Error("Error decoding JSON: %v", err)
			httpBodyFinal = responseBody.String()
		} else {
			if tokenResponse.OK {
				httpBodyFinal = "success"
			} else {
				httpBodyFinal = responseBody.String()
			}
		}
	} else {
		httpBodyFinal = responseBody.String()
	}

	if httpStatusCode > 0 && httpStatusCode < 400 {
		slog.Info("TELEGRAM", "response", httpBodyFinal)
	} else {
		slog.Info("TELEGRAM", "response", httpBodyFinal)
	}
}
