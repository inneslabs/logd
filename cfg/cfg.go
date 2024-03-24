package cfg

import (
	"fmt"
	"os"

	"github.com/inneslabs/logd/store"
	"gopkg.in/yaml.v3"
)

type LogdCfg struct {
	UdpLaddrPort             string     `yaml:"udp_laddr_port"` // string supports fly-global-services:6102
	AppPort                  int        `yaml:"app_port"`
	ReadSecret               string     `yaml:"read_secret"`
	WriteSecret              string     `yaml:"write_secret"`
	AccessControlAllowOrigin string     `yaml:"access_control_allow_origin"`
	Store                    *store.Cfg `yaml:"store"`
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

	if cfg.ReadSecret != "" || cfg.WriteSecret != "" {
		fmt.Println("warning: in production, do not set read_secret or write_secret in the logdrc.yml")
	}

	return nil
}
