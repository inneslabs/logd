package cfg

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

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

// Recursively tries to load config by filename, from dir up to the root
func Load(fname, dir string, cfg *LogdCfg) error {
	file, err := os.OpenFile(path.Join(dir, fname), os.O_RDONLY, 0777)
	if err != nil {
		// up to the root
		parentDir := filepath.Dir(dir)
		if dir != parentDir {
			return Load(fname, parentDir, cfg)
		}
		return fmt.Errorf("err opening file: %w", err)
	}
	defer file.Close()

	fmt.Printf("found cfg file \"%s\" in %s, decoding yaml", fname, dir)

	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	err = dec.Decode(&cfg)
	if err != nil {
		return fmt.Errorf("err decoding cfg file (%s): %w", fname, err)
	}

	return nil
}
