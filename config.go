package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type programOptions struct {
	Runtime struct {
		RuntimeDir    string `toml:"runtime_dir"`
		ServerAddress string `toml:"server_address"`
		ClientProto   string `toml:"client_proto"`
		Provider      string `toml:"provider"`
	} `toml:"runtime"`

	// Protocol settings.
	ProtobufClient struct {
		ServerKey string `toml:"server_key"`
		PeerKey   string `toml:"peer_key"`
	} `toml:"client_protobuf"`

	// Provider settings.
	LinodeParams struct {
		AccessToken string `toml:"access_token"`
		Region      string `toml:"region"`
		Plan        string `toml:"plan"`
	} `toml:"provider_linode"`
	DigitalOceanParams struct {
		AccessToken string `toml:"access_token"`
		Region      string `toml:"region"`
		Plan        string `toml:"plan"`
	} `toml:"provider_digitalocean"`

	// User settings.
	AllUsers struct {
		SSHKeys []string `toml:"ssh_keys"`
	} `toml:"user_common"`
	RootUser struct {
		Password string `toml:"password"`
	} `toml:"user_root"`
	NormalUser struct {
		UserName string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"user_unpriv"`

	// Circumvention method settings.
	WireGuard struct {
		Enable    bool     `toml:"enable"`
		ServerKey string   `toml:"server_key"`
		PeerKeys  []string `toml:"peer_keys"`
		Port      uint     `toml:"port"`
	} `toml:"wireguard"`
	ObfsproxyIPv4 struct {
		Enable bool   `toml:"enable"`
		Secret string `toml:"secret"`
		Port   uint   `toml:"port"`
	} `toml:"obfsproxy_ipv4"`
	ObfsproxyIPv6 struct {
		Enable bool   `toml:"enable"`
		Secret string `toml:"secret"`
		Port   uint   `toml:"port"`
	} `toml:"obfsproxy_ipv6"`
}

func newProgramOptions(filename string) (*programOptions, error) {
	if len(filename) == 0 {
		log.Error("Config path is empty or missing")
		return nil, fmt.Errorf("missing config path")
	}

	var config programOptions
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		log.WithFields(log.Fields{
			"cause": err,
			"path":  filename,
		}).Error("Error reading config file")
		return nil, err
	}

	// Substitute variables in RuntimeDir path.
	if strings.Contains(config.Runtime.RuntimeDir, "${HOME}") {
		user, err := user.Current()
		if err != nil {
			log.WithFields(log.Fields{
				"cause": err,
				"path":  filename,
			}).Error("Unable to retrieve current user information when trying " +
				"to substitute ${HOME} with path to home dir")
			return nil, err
		}
		config.Runtime.RuntimeDir = strings.Replace(config.Runtime.RuntimeDir,
			"${HOME}", user.HomeDir, -1)
	}

	if strings.Contains(config.Runtime.RuntimeDir, "${EXE}") {
		exePath, err := os.Executable()
		if err != nil {
			log.WithFields(log.Fields{
				"cause": err,
				"path":  filename,
			}).Error("Unable to retrieve path to program executable when trying " +
				"to substitute ${EXE}")
			return nil, err
		}
		config.Runtime.RuntimeDir = strings.Replace(config.Runtime.RuntimeDir,
			"${EXE}", path.Dir(exePath), -1)
	}

	if strings.Contains(config.Runtime.RuntimeDir, "${AUTO}") {
		panic("runtime.runtime_dir ${AUTO} substitution is not implemented yet.")
	}

	// Generate random service ports, if needed.
	if config.WireGuard.Enable && config.WireGuard.Port == 0 {
		config.WireGuard.Port = randomPort()
	}
	if config.ObfsproxyIPv4.Enable && config.ObfsproxyIPv4.Port == 0 {
		config.ObfsproxyIPv4.Port = randomPort()
	}
	if config.ObfsproxyIPv6.Enable && config.ObfsproxyIPv6.Port == 0 {
		config.ObfsproxyIPv6.Port = randomPort()
	}
	return &config, nil
}

func logConfigurationError(cause string, extra ...log.Fields) error {
	fields := log.Fields{
		"cause": cause,
	}
	if len(extra) > 0 {
		for k, v := range extra[0] {
			fields[k] = v
		}
	}
	log.WithFields(fields).Error("Configuration error")
	return errors.New("configuration error")
}

func validateGeneralProgramOptions(o *programOptions) error {
	// TODO: Not all settings are validated.

	// Validate runtime settings.
	{
		_, err := url.Parse(o.Runtime.ServerAddress)
		if err != nil {
			cause := fmt.Sprintf("runtime: malformed server address: %s", err.Error())
			return logConfigurationError(cause)
		}

		// TODO: validate provider name.
		if len(o.Runtime.Provider) == 0 {
			return logConfigurationError("runtime: provider is empty or missing")
		}
		if len(o.Runtime.RuntimeDir) == 0 {
			return logConfigurationError("runtime: runtime dir is empty or missing")
		}
	}

	// Wireguard section.
	if o.WireGuard.Enable {
		if len(o.WireGuard.ServerKey) == 0 {
			return logConfigurationError("wireguard: server key is empty or missing")
		}
		if len(o.WireGuard.PeerKeys) == 0 {
			return logConfigurationError("wireguard: at least 1 peer key is required")
		}
		if o.WireGuard.Port == 0 || o.WireGuard.Port > 65535 {
			return logConfigurationError("wireguard: missing or invalid port number")
		}
	}

	// Obfsproxy IPv4 section.
	if o.ObfsproxyIPv4.Enable {
		if len(o.ObfsproxyIPv4.Secret) == 0 {
			return logConfigurationError("obfsproxy ipv4: missing secret")
		}
		if o.ObfsproxyIPv4.Port == 0 || o.ObfsproxyIPv4.Port > 65535 {
			return logConfigurationError("obfsproxy ipv4: missing or invalid port number")
		}
	}

	// Obfsproxy IPv6 section.
	if o.ObfsproxyIPv6.Enable {
		if len(o.ObfsproxyIPv6.Secret) == 0 {
			return logConfigurationError("obfsproxy ipv6: missing secret")
		}
		if o.ObfsproxyIPv6.Port == 0 || o.ObfsproxyIPv6.Port > 65535 {
			return logConfigurationError("obfsproxy ipv6: missing or invalid port number")
		}
	}
	return nil
}

func randomPort() uint {
	bigInt, err := rand.Int(rand.Reader, big.NewInt(54000))
	if err != nil {
		panic("Random generator error: " + err.Error())
	}
	return uint(10000 + int(bigInt.Int64()))
}
