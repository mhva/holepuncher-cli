package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type providerType int

const (
	providerTypeLinode providerType = iota
	providerTypeDigitalOcean
)

type tunnelCreationParams struct {
	WireGuardEnabled   bool     `json:"wireguard_enabled"`
	WireGuardServerKey string   `json:"wireguard_server_key,omitempty"`
	WireGuardPeerKeys  []string `json:"wireguard_peer_keys,omitempty"`
	WireGuardPort      uint     `json:"wireguard_port,omitempty"`

	ObfsproxyIPv4Enabled bool   `json:"obfsproxy4_enabled"`
	ObfsproxyIPv4Secret  string `json:"obfsproxy4_secret,omitempty"`
	ObfsproxyIPv4Port    uint   `json:"obfsproxy4_port,omitempty"`

	ObfsproxyIPv6Enabled bool   `json:"obfsproxy6_enabled"`
	ObfsproxyIPv6Secret  string `json:"obfsproxy6_secret,omitempty"`
	ObfsproxyIPv6Port    uint   `json:"obfsproxy6_port,omitempty"`
}

type tunnelInstance struct {
	Provider  providerType `json:"provider"`
	Label     string       `json:"label"`
	IPv4      []string     `json:"ipv4"`
	IPv6      []string     `json:"ipv6"`
	CreatedAt time.Time    `json:"created_at"`
}

type createTunnelResult struct {
	CreationParams tunnelCreationParams
	Instance       tunnelInstance
}

type rebuildTunnelResult struct {
	CreationParams tunnelCreationParams
	Instance       tunnelInstance
}

type aCloudProvider interface {
	CreateTunnel() (*createTunnelResult, error)
	TunnelStatus() (*tunnelInstance, error)
	DestroyTunnel() error
}

func (p providerType) String() string {
	switch p {
	case providerTypeLinode:
		return "linode"
	case providerTypeDigitalOcean:
		return "digital_ocean"
	default:
		return fmt.Sprintf("%d (unsupported)", p)
	}
}

func newHTTPClient(_ *programOptions) *http.Client {
	return &http.Client{
		Timeout: 150 * time.Second,
	}
}

func newCloudProviderFromOptions(options *programOptions) (aCloudProvider, error) {
	client, err := newHolepuncherClient(options, newHTTPClient(options))
	if err != nil {
		return nil, err
	}

	switch options.Runtime.Provider {
	case providerTypeLinode.String():
		return newLinodeProvider(client, options)
	default:
		log.WithField("provider", options.Runtime.Provider).Error("Provider is not supported")
		return nil, errors.New("unsupported provider")
	}
}

func creationParamsFromProgramOptions(options *programOptions) tunnelCreationParams {
	params := tunnelCreationParams{}
	if options.WireGuard.Enable {
		params.WireGuardEnabled = options.WireGuard.Enable
		params.WireGuardServerKey = options.WireGuard.ServerKey
		params.WireGuardPeerKeys = options.WireGuard.PeerKeys
		params.WireGuardPort = options.WireGuard.Port
	}
	if options.ObfsproxyIPv4.Enable {
		params.ObfsproxyIPv4Enabled = options.ObfsproxyIPv4.Enable
		params.ObfsproxyIPv4Secret = options.ObfsproxyIPv4.Secret
		params.ObfsproxyIPv4Port = options.ObfsproxyIPv4.Port
	}
	if options.ObfsproxyIPv6.Enable {
		params.ObfsproxyIPv6Enabled = options.ObfsproxyIPv6.Enable
		params.ObfsproxyIPv6Secret = options.ObfsproxyIPv6.Secret
		params.ObfsproxyIPv6Port = options.ObfsproxyIPv6.Port
	}
	return params
}
