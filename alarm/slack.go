package alarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackMsg struct {
	Text string `json:"text"`
}

func SendSlackMsg(msg, slackWebhook string) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(&SlackMsg{
		Text: msg,
	})
	if err != nil {
		return fmt.Errorf("marshal json err: %w", err)
	}
	resp, err := http.Post(slackWebhook, "application/json", buf)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("failed to send slack message: %d %s %w", resp.StatusCode, resp.Status, err)
	}
	return nil
}
