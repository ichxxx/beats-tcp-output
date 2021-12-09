package tcp

import (
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/pkg/errors"
)

type Config struct {
	Host            string       `config:"host"`
	Port            string       `config:"port"`

	BufferSize      int          `config:"buffer_size"`
	WritevEnable    bool         `config:"writev"`
	SSLEnable       bool         `config:"ssl.enable"`
	SSLCertPath     string       `config:"ssl.cert_path"`
	SSLKeyPath      string       `config:"ssl.key_path"`

	lineDelimiter   string       `config:"line_delimiter"`
	Codec           codec.Config `config:"codec"`
}

var defaultConfig = Config{
	BufferSize: 1 << 15,
	WritevEnable: true,
	lineDelimiter: "\n",
}

func (c *Config) Validate() error {
	if c.SSLEnable {
		if _, err := os.Stat(c.SSLCertPath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("Certificate %s not found", c.SSLCertPath))
		}
		if _, err := os.Stat(c.SSLKeyPath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("Key %s not found", c.SSLKeyPath))
		}
	}
	return nil
}
