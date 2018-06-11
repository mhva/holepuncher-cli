package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type erasedLinodeRPCFn func(*providerLinode) (interface{}, error)

// prettyPrint prints given value as indented JSON.
func prettyPrint(v interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(v); err != nil {
		log.Warning("Unable to pretty-print tunnel information, will dump raw data instead...")
		fmt.Printf("%+v\n", v)
	}
}

func doLinodeRPC(c *cli.Context, fn erasedLinodeRPCFn) (interface{}, error) {
	options, err := newProgramOptions(c.GlobalString("config"))
	if err != nil {
		return nil, err
	}

	client, err := newHolepuncherClient(options, newHTTPClient(options))
	if err != nil {
		return nil, err
	}
	provider, err := newLinodeProvider(client, options)
	if err != nil {
		return nil, err
	}
	return fn(provider)
}

func printLinodeResult(c *cli.Context, fn erasedLinodeRPCFn) error {
	result, err := doLinodeRPC(c, fn)
	if err != nil {
		return err
	}
	prettyPrint(result)
	return nil
}

func newCloudProviderFromContext(c *cli.Context) (aCloudProvider, *programOptions, error) {
	options, err := newProgramOptions(c.GlobalString("config"))
	if err != nil {
		return nil, nil, err
	}
	provider, err := newCloudProviderFromOptions(options)
	if err != nil {
		return nil, nil, err
	}
	return provider, options, err
}

func handleCreateTunnelCommand(c *cli.Context) error {
	provider, options, err := newCloudProviderFromContext(c)
	if err != nil {
		return err
	}
	result, err := provider.CreateTunnel()
	if err != nil {
		return err
	}
	log.Info("Tunnel instance was successfully created")

	cache := &sessionCache{
		InstanceInfo:   &result.Instance,
		CreationParams: &result.CreationParams,
	}
	saveSessionCache(cache, options.Runtime.RuntimeDir)
	return nil
}

func handleDestroyTunnelCommand(c *cli.Context) error {
	provider, options, err := newCloudProviderFromContext(c)
	if err != nil {
		return err
	}
	if err = provider.DestroyTunnel(); err != nil {
		return err
	}
	log.Info("Tunnel instance was successfully deleted")

	// Remove session cache because as of now it is invalid.
	clearSessionCache(options.Runtime.RuntimeDir)
	return nil
}

func handleShowTunnelInfoCommand(c *cli.Context) error {
	provider, _, err := newCloudProviderFromContext(c)
	if err != nil {
		return err
	}
	result, err := provider.TunnelStatus()
	if err != nil {
		return err
	}

	prettyPrint(result)
	return nil
}

func handlePrintSessionVarCommand(c *cli.Context, sessionVar string) error {
	options, err := newProgramOptions(c.GlobalString("config"))
	if err != nil {
		return err
	}
	session, err := restoreSessionCache(options.Runtime.RuntimeDir)
	if err != nil {
		return err
	}

	switch sessionVar {
	case "ipv4":
		fmt.Println(strings.Join(session.InstanceInfo.IPv4, "\n"))
	case "ipv6":
		fmt.Println(strings.Join(session.InstanceInfo.IPv6, "\n"))
	case "created":
		fmt.Println(session.InstanceInfo.CreatedAt.String())
	case "duration":
		fmt.Println(time.Since(session.InstanceInfo.CreatedAt).String())

	case "wg.enabled":
		fmt.Printf("%t\n", session.CreationParams.WireGuardEnabled)
	case "wg.server_key":
		fmt.Println(session.CreationParams.WireGuardServerKey)
	case "wg.peer_keys":
		fmt.Println(strings.Join(session.CreationParams.WireGuardPeerKeys, "\n"))
	case "wg.port":
		fmt.Printf("%d\n", session.CreationParams.WireGuardPort)

	case "obfs4.enabled":
		fmt.Printf("%t\n", session.CreationParams.ObfsproxyIPv4Enabled)
	case "obfs4.secret":
		fmt.Println(session.CreationParams.ObfsproxyIPv4Secret)
	case "obfs4.port":
		fmt.Printf("%d\n", session.CreationParams.ObfsproxyIPv4Port)

	case "obfs6.enabled":
		fmt.Printf("%t\n", session.CreationParams.ObfsproxyIPv6Enabled)
	case "obfs6.secret":
		fmt.Println(session.CreationParams.ObfsproxyIPv6Secret)
	case "obfs6.port":
		fmt.Printf("%d\n", session.CreationParams.ObfsproxyIPv6Port)
	}
	return nil
}

func handleRebuildLinodeTunnel(c *cli.Context) error {
	// FIXME: creating programOptions twice (here and within doLinodeRPC).
	options, err := newProgramOptions(c.GlobalString("config"))
	if err != nil {
		return err
	}

	fn := func(p *providerLinode) (interface{}, error) {
		return p.RebuildTunnel()
	}
	result, err := doLinodeRPC(c, fn)
	if err != nil {
		return err
	}

	info := result.(*rebuildTunnelResult)
	cache := &sessionCache{
		InstanceInfo:   &info.Instance,
		CreationParams: &info.CreationParams,
	}

	saveSessionCache(cache, options.Runtime.RuntimeDir)
	return nil
}

func handleListLinodeInstances(c *cli.Context) error {
	fn := func(p *providerLinode) (interface{}, error) {
		return p.ListInstances()
	}
	return printLinodeResult(c, fn)
}

func handleListLinodePlans(c *cli.Context) error {
	fn := func(p *providerLinode) (interface{}, error) {
		return p.ListPlans()
	}
	return printLinodeResult(c, fn)
}

func handleListLinodeRegions(c *cli.Context) error {
	fn := func(p *providerLinode) (interface{}, error) {
		return p.ListRegions()
	}
	return printLinodeResult(c, fn)
}

func handleListLinodeImages(c *cli.Context) error {
	fn := func(p *providerLinode) (interface{}, error) {
		return p.ListImages()
	}
	return printLinodeResult(c, fn)
}

func handleListLinodeStackScripts(c *cli.Context) error {
	fn := func(p *providerLinode) (interface{}, error) {
		return p.ListStackScripts()
	}
	return printLinodeResult(c, fn)
}

func main() {
	app := cli.NewApp()
	app.Name = "holepuncher-cli"
	app.Usage = "holepuncher client"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "config file",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "create",
			Usage:  "create tunnel",
			Action: handleCreateTunnelCommand,
		},
		{
			Name:   "destroy",
			Usage:  "destroy tunnel",
			Action: handleDestroyTunnelCommand,
		},
		{
			Name:   "info",
			Usage:  "display tunnel info",
			Action: handleShowTunnelInfoCommand,
		},
		{
			Name:  "linode",
			Usage: "linode-specific actions",
			Subcommands: []cli.Command{
				{
					Name:   "rebuild",
					Usage:  "rebuilds tunnel",
					Action: handleRebuildLinodeTunnel,
				},
				{
					Name:   "instances",
					Usage:  "list currently active instances",
					Action: handleListLinodeInstances,
				},
				{
					Name:   "plans",
					Usage:  "list available instance types",
					Action: handleListLinodePlans,
				},
				{
					Name:   "regions",
					Usage:  "list available regions",
					Action: handleListLinodeRegions,
				},
				{
					Name:   "images",
					Usage:  "list available images",
					Action: handleListLinodeImages,
				},
				{
					Name:   "stackscripts",
					Usage:  "list available StackScripts",
					Action: handleListLinodeStackScripts,
				},
			},
		},
		{
			Name:  "var",
			Usage: "print variable from current session",
			Subcommands: []cli.Command{
				{
					Name:  "ipv4",
					Usage: "list of ipv4 addresses separated by newline (LF)",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "ipv4")
					},
				},
				{
					Name:  "ipv6",
					Usage: "list of ipv6 addresses separated by newline (LF)",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "ipv6")
					},
				},
				{
					Name:  "created",
					Usage: "creation date",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "created")
					},
				},
				{
					Name:  "duration",
					Usage: "tunnel lifetime since creation",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "duration")
					},
				},
				{
					Name:  "wg.enabled",
					Usage: "wireguard state (true/false)",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "wg.enabled")
					},
				},
				{
					Name:  "wg.server_key",
					Usage: "wireguard server key",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "wg.server_key")
					},
				},
				{
					Name:  "wg.peer_keys",
					Usage: "list of wireguard peer keys",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "wg.peer_keys")
					},
				},
				{
					Name:  "wg.port",
					Usage: "wireguard port number",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "wg.port")
					},
				},
				{
					Name:  "obfs4.enabled",
					Usage: "obfsproxy ipv4 state (true/false)",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "obfs4.enabled")
					},
				},
				{
					Name:  "obfs4.secret",
					Usage: "obfsproxy ipv4 secret",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "obfs4.secret")
					},
				},
				{
					Name:  "obfs4.port",
					Usage: "obfsproxy ipv4 port number",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "obfs4.port")
					},
				},
				{
					Name:  "obfs6.enabled",
					Usage: "obfsproxy ipv6 state (true/false)",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "obfs6.enabled")
					},
				},
				{
					Name:  "obfs6.secret",
					Usage: "obfsproxy ipv6 secret",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "obfs6.secret")
					},
				},
				{
					Name:  "obfs6.port",
					Usage: "obfsproxy ipv6 port number",
					Action: func(c *cli.Context) error {
						return handlePrintSessionVarCommand(c, "obfs6.port")
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
