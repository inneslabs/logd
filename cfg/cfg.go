package cfg

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type LogdCfg struct {
	UdpLaddrPort             string // string supports fly-global-services:6102
	AppPort                  int
	ReadSecret               string
	WriteSecret              string
	AccessControlAllowOrigin string
}

func Load(fname string, cfg *LogdCfg) error {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0777)
	if err != nil {
		return fmt.Errorf("err opening file: %w", err)
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	err = dec.Decode(&cfg)
	if err != nil {
		return fmt.Errorf("err decoding cfg file (%s): %w", fname, err)
	}

	return nil
}
