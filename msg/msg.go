package msg

type Msg struct {
	Timestamp int64  `json:"t"`
	Env       string `json:"env"`
	Svc       string `json:"svc"`
	Fn        string `json:"fn"`
	Lvl       string `json:"lvl"`
	Msg       string `json:"msg"`
	Dump      string `json:"dump,omitempty"`
}
