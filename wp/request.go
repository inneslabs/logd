package wp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Request struct {
	Raw            string `json:"-"`
	Env            string `json:"-"`
	Id             string `json:"request_id"`
	Timestamp      string `json:"timestamp_iso8601"`
	Method         string `json:"request_type"`
	Url            string `json:"request_url"`
	ResponseStatus string `json:"status"`
}

func DecodeRequest(raw, env string) (*Request, error) {
	buf := new(bytes.Buffer)
	buf.WriteString(raw)
	dec := json.NewDecoder(buf)
	req := &Request{
		Raw: raw,
		Env: env,
	}
	err := dec.Decode(req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request json: %w", err)
	}
	return req, nil
}

func (r *Request) ParsedTimestamp() (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05+00:00", r.Timestamp)
}
