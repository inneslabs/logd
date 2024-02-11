/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package alarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type slackMsg struct {
	Text string `json:"text"`
}

func SendSlackMsg(msg, slackWebhook string) error {
	fmt.Println("sending slack message to ", slackWebhook, msg)
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(&slackMsg{msg})
	if err != nil {
		return fmt.Errorf("marshal json err: %w", err)
	}
	resp, err := http.Post(slackWebhook, "application/json", buf)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send slack message: %d %s", resp.StatusCode, resp.Status)
	}
	return nil
}
