package main

import (
	"protoapi"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type providerLinode struct {
	client  aHolepuncherClient
	options *programOptions
}

type linodeInstance struct {
	ID         int64  `json:"id"`
	Label      string `json:"label"`
	Group      string
	Region     string
	Plan       string
	Image      string
	Status     string
	IPv4       []string
	IPv6       []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Hypervisor string
	Disk       uint64
	Memory     uint64
	VCPUs      uint
	Transfer   uint64
}

type linodePlan struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	Class        string  `json:"class,omitempty"`
	PriceHourly  float32 `json:"price_hourly"`
	PriceMonthly float32 `json:"price_monthly"`
	Memory       uint64  `json:"memory,omitempty"`
	Bandwidth    uint64  `json:"bandwidth,omitempty"`
	Transfer     uint64  `json:"transfer,omitempty"`
	Vcpus        uint    `json:"vcpus,omitempty"`
}

type linodeRegion struct {
	ID      string `json:"id"`
	Country string `json:"country"`
}

type linodeImage struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Size      uint64    `json:"size"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	Vendor    string    `json:"vendor"`
}

type linodeStackScript struct {
	ID          int64  `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

func newLinodeProvider(client aHolepuncherClient, opts *programOptions) (*providerLinode, error) {
	// General validation should ensure that most of the data is valid, but
	// it does not check provider parameters.
	if err := validateGeneralProgramOptions(opts); err != nil {
		return nil, err
	}

	if len(opts.LinodeParams.AccessToken) == 0 {
		return nil, logConfigurationError("linode: access token is empty or missing")
	} else if len(opts.LinodeParams.Plan) == 0 {
		return nil, logConfigurationError("linode: plan is empty or missing")
	} else if len(opts.LinodeParams.Region) == 0 {
		return nil, logConfigurationError("linode: region is empty or missing")
	}

	return &providerLinode{
		client:  client,
		options: opts,
	}, nil
}

func (p *providerLinode) CreateTunnel() (*createTunnelResult, error) {
	generic, err := p.request(p.createCreateTunnelRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeCreateTunnelResult()
	if result == nil {
		log.Error("Expected LinodeCreateTunnelResponse, got something else (BUG)")
		return nil, errors.New("LinodeCreateTunnel RPC bug")
	} else if linodeErr := result.GetError(); linodeErr != nil {
		p.logError("RPC method returned an error", linodeErr)
		return nil, errors.New("rpc method returned an error")
	} else if result.GetInstance() == nil {
		// Should be unreachable unless there's a bug in the server code.
		log.Error("Both result and error objects are empty (BUG)")
		return nil, errors.New("LinodeCreateTunnel RPC bug")
	}

	p.logInstance(result.GetInstance(), "Successfully created Linode instance")

	createdAt, _ := p.parseDate(result.GetInstance().CreatedAt)
	instance := tunnelInstance{
		Provider:  providerTypeLinode,
		Label:     result.GetInstance().Label,
		IPv4:      result.GetInstance().Ipv4,
		IPv6:      result.GetInstance().Ipv6,
		CreatedAt: createdAt,
	}

	return &createTunnelResult{
		CreationParams: creationParamsFromProgramOptions(p.options),
		Instance:       instance,
	}, nil
}

func (p *providerLinode) RebuildTunnel() (*rebuildTunnelResult, error) {
	generic, err := p.request(p.createRebuildTunnelRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeRebuildTunnelResult()
	if result == nil {
		log.Error("Expected LinodeRebuildTunnelResponse, got something else (BUG)")
		return nil, errors.New("LinodeRebuildTunnel RPC bug")
	} else if linodeErr := result.GetError(); linodeErr != nil {
		p.logError("RPC method returned an error", linodeErr)
		return nil, errors.New("rpc method returned an error")
	} else if result.GetInstance() == nil {
		// Should be unreachable unless there's a bug in the server code.
		log.Error("Both result and error objects are empty (BUG)")
		return nil, errors.New("LinodeRebuildTunnel RPC bug")
	}

	p.logInstance(result.GetInstance(), "Successfully rebuilt Linode instance")

	createdAt, _ := p.parseDate(result.GetInstance().CreatedAt)
	instance := tunnelInstance{
		Provider:  providerTypeLinode,
		Label:     result.GetInstance().Label,
		IPv4:      result.GetInstance().Ipv4,
		IPv6:      result.GetInstance().Ipv6,
		CreatedAt: createdAt,
	}

	return &rebuildTunnelResult{
		CreationParams: creationParamsFromProgramOptions(p.options),
		Instance:       instance,
	}, nil
}

func (p *providerLinode) DestroyTunnel() error {
	generic, err := p.request(p.createDestroyTunnelRequest())
	if err != nil {
		return err
	}

	result := generic.GetLinodeDestroyTunnelResult()
	if result == nil {
		log.Error("Expected GetLinodeDestroyTunnelResponse, got something else (BUG)")
		return errors.New("LinodeDestroyTunnel RPC bug")
	} else if linodeErr := result.GetError(); linodeErr != nil {
		p.logError("RPC method returned an error", linodeErr)
		return errors.New("rpc method returned an error")
	}

	return nil
}

func (p *providerLinode) TunnelStatus() (*tunnelInstance, error) {
	generic, err := p.request(p.createTunnelStatusRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeTunnelStatusResult()
	if result == nil {
		log.Error("Expected LinodeGetTunnelStatusResponse, but got something else (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	} else if result.GetError() != nil {
		p.logError("RPC method returned an error", result.GetError())
		return nil, errors.New("rpc method returned an error")
	} else if result.GetInstance() == nil {
		log.Error("Result and error objects are both empty (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	}

	createdAt, _ := p.parseDate(result.GetInstance().CreatedAt)
	return &tunnelInstance{
		Provider:  providerTypeLinode,
		Label:     result.GetInstance().Label,
		IPv4:      result.GetInstance().Ipv4,
		IPv6:      result.GetInstance().Ipv6,
		CreatedAt: createdAt,
	}, nil
}

func (p *providerLinode) ListInstances() ([]*linodeInstance, error) {
	generic, err := p.request(p.createListInstancesRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeListInstancesResult()
	if result == nil {
		log.Error("Expected LinodeListInstancesResponse, but got something else (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	} else if result.GetError() != nil {
		p.logError("RPC method returned an error", result.GetError())
		return nil, errors.New("rpc method returned an error")
	} else if result.GetInstances() == nil {
		log.Error("Result and error objects are both empty (BUG)")
		return nil, errors.New("LinodeListInstancesResponse RPC bug")
	}

	instances := []*linodeInstance{}
	for _, instance := range result.GetInstances().GetL() {
		createdAt, _ := p.parseDate(instance.CreatedAt)
		updatedAt, _ := p.parseDate(instance.UpdatedAt)
		instances = append(instances, &linodeInstance{
			ID:         instance.Id,
			Label:      instance.Label,
			Group:      instance.Group,
			Region:     instance.Region,
			Plan:       instance.Plan,
			Image:      instance.Image,
			Status:     strings.ToLower(instance.Status.String()),
			IPv4:       instance.Ipv4,
			IPv6:       instance.Ipv6,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			Hypervisor: instance.Hypervisor,
			Disk:       instance.Disk,
			Memory:     instance.Memory,
			Transfer:   instance.Transfer,
			VCPUs:      uint(instance.Vcpus),
		})
	}
	return instances, nil
}

func (p *providerLinode) ListPlans() ([]*linodePlan, error) {
	generic, err := p.request(p.createListPlansRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeListPlansResult()
	if result == nil {
		log.Error("Expected LinodeListPlansResponse, but got something else (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	} else if result.GetError() != nil {
		p.logError("RPC method returned an error", result.GetError())
		return nil, errors.New("rpc method returned an error")
	} else if result.GetPlans() == nil {
		log.Error("Result and error objects are both empty (BUG)")
		return nil, errors.New("LinodeListPlansResponse RPC bug")
	}

	plans := []*linodePlan{}
	for _, plan := range result.GetPlans().GetL() {
		plans = append(plans, &linodePlan{
			ID:           plan.Id,
			Label:        plan.Label,
			Class:        plan.Class,
			PriceHourly:  plan.PriceHourly,
			PriceMonthly: plan.PriceMonthly,
			Memory:       plan.Memory,
			Bandwidth:    plan.NetworkOut,
			Transfer:     plan.Transfer,
			Vcpus:        uint(plan.Vcpus),
		})
	}
	return plans, nil
}

func (p *providerLinode) ListRegions() ([]*linodeRegion, error) {
	generic, err := p.request(p.createListRegionsRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeListRegionsResult()
	if result == nil {
		log.Error("Expected LinodeListRegionsResponse, but got something else (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	} else if result.GetError() != nil {
		p.logError("RPC method returned an error", result.GetError())
		return nil, errors.New("rpc method returned an error")
	} else if result.GetRegions() == nil {
		log.Error("Result and error objects are both empty (BUG)")
		return nil, errors.New("LinodeListRegionsResponse RPC bug")
	}

	regions := []*linodeRegion{}
	for _, region := range result.GetRegions().GetL() {
		regions = append(regions, &linodeRegion{
			ID:      region.Id,
			Country: region.Country,
		})
	}
	return regions, nil
}

func (p *providerLinode) ListImages() ([]*linodeImage, error) {
	generic, err := p.request(p.createListImagesRequest())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeListImagesResult()
	if result == nil {
		log.Error("Expected LinodeListImagesResponse, but got something else (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	} else if result.GetError() != nil {
		p.logError("RPC method returned an error", result.GetError())
		return nil, errors.New("rpc method returned an error")
	} else if result.GetImages() == nil {
		log.Error("Result and error objects are both empty (BUG)")
		return nil, errors.New("LinodeListImagesResponse RPC bug")
	}

	images := []*linodeImage{}
	for _, image := range result.GetImages().GetL() {
		createdAt, _ := p.parseDate(image.CreatedAt)
		images = append(images, &linodeImage{
			ID:        image.Id,
			Label:     image.Label,
			Size:      image.Size,
			CreatedBy: image.CreatedBy,
			CreatedAt: createdAt,
			Vendor:    image.Vendor,
		})
	}
	return images, nil
}

func (p *providerLinode) ListStackScripts() ([]*linodeStackScript, error) {
	generic, err := p.request(p.createListStackScripts())
	if err != nil {
		return nil, err
	}

	result := generic.GetLinodeListStackscriptsResult()
	if result == nil {
		log.Error("Expected LinodeListStackScriptsResponse, but got something else (BUG)")
		return nil, errors.New("LinodeGetTunnelStatus RPC bug")
	} else if result.GetError() != nil {
		p.logError("RPC method returned an error", result.GetError())
		return nil, errors.New("rpc method returned an error")
	} else if result.GetStackscripts() == nil {
		log.Error("Result and error objects are both empty (BUG)")
		return nil, errors.New("LinodeListStackScriptsResponse RPC bug")
	}

	scripts := []*linodeStackScript{}
	for _, script := range result.GetStackscripts().GetL() {
		scripts = append(scripts, &linodeStackScript{
			ID:          script.Id,
			Label:       script.Label,
			Description: script.Description,
			Body:        script.Body,
		})
	}
	return scripts, nil
}

func (p *providerLinode) request(m *protoapi.Request) (*protoapi.Response, error) {
	generic, err := p.client.DoRequest(m)
	if err != nil {
		log.WithField("cause", err).Error("Fundamental RPC failure")
		return nil, errors.Wrap(err, "fundamental rpc failure")
	}
	return generic, nil
}

func (p *providerLinode) createAuth() *protoapi.LinodeAuth {
	return &protoapi.LinodeAuth{
		AccessToken: p.options.LinodeParams.AccessToken,
	}
}

func (p *providerLinode) netServicesOptions() (
	*protoapi.WireguardOptions,
	*protoapi.ObfsproxyIPv4Options,
	*protoapi.ObfsproxyIPv6Options,
) {
	var wireguardOptions *protoapi.WireguardOptions
	var obfs4Options *protoapi.ObfsproxyIPv4Options
	var obfs6Options *protoapi.ObfsproxyIPv6Options
	if p.options.WireGuard.Enable {
		wireguardOptions = &protoapi.WireguardOptions{
			Port:      uint32(p.options.WireGuard.Port),
			ServerKey: p.options.WireGuard.ServerKey,
			PeerKeys:  p.options.WireGuard.PeerKeys,
		}
	}
	if p.options.ObfsproxyIPv4.Enable {
		obfs4Options = &protoapi.ObfsproxyIPv4Options{
			Port:   uint32(p.options.ObfsproxyIPv4.Port),
			Secret: p.options.ObfsproxyIPv4.Secret,
		}
	}
	if p.options.ObfsproxyIPv6.Enable {
		obfs6Options = &protoapi.ObfsproxyIPv6Options{
			Port:   uint32(p.options.ObfsproxyIPv6.Port),
			Secret: p.options.ObfsproxyIPv6.Secret,
		}
	}
	return wireguardOptions, obfs4Options, obfs6Options
}

func (p *providerLinode) createCreateTunnelRequest() *protoapi.Request {
	wg, obfs4, obfs6 := p.netServicesOptions()
	command := &protoapi.LinodeCreateTunnelRequest{
		Auth:                   p.createAuth(),
		Region:                 p.options.LinodeParams.Region,
		Plan:                   p.options.LinodeParams.Plan,
		RootPassword:           p.options.RootUser.Password,
		RegularAccountName:     p.options.NormalUser.UserName,
		RegularAccountPassword: p.options.NormalUser.Password,
		SshKeys:                p.options.AllUsers.SSHKeys,
		WireguardOptions:       wg,
		Obfsproxy4Options:      obfs4,
		Obfsproxy6Options:      obfs6,
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeCreateTunnel{LinodeCreateTunnel: command},
	}
}

func (p *providerLinode) createRebuildTunnelRequest() *protoapi.Request {
	wg, obfs4, obfs6 := p.netServicesOptions()
	command := &protoapi.LinodeRebuildTunnelRequest{
		Auth:                   p.createAuth(),
		RootPassword:           p.options.RootUser.Password,
		RegularAccountName:     p.options.NormalUser.UserName,
		RegularAccountPassword: p.options.NormalUser.Password,
		SshKeys:                p.options.AllUsers.SSHKeys,
		WireguardOptions:       wg,
		Obfsproxy4Options:      obfs4,
		Obfsproxy6Options:      obfs6,
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeRebuildTunnel{LinodeRebuildTunnel: command},
	}
}

func (p *providerLinode) createDestroyTunnelRequest() *protoapi.Request {
	command := &protoapi.LinodeDestroyTunnelRequest{
		Auth: p.createAuth(),
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeDestroyTunnel{LinodeDestroyTunnel: command},
	}
}

func (p *providerLinode) createTunnelStatusRequest() *protoapi.Request {
	command := &protoapi.LinodeGetTunnelStatusRequest{
		Auth: p.createAuth(),
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeTunnelStatus{LinodeTunnelStatus: command},
	}
}

func (p *providerLinode) createListPlansRequest() *protoapi.Request {
	command := &protoapi.LinodeListPlansRequest{}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeListPlans{LinodeListPlans: command},
	}
}

func (p *providerLinode) createListRegionsRequest() *protoapi.Request {
	command := &protoapi.LinodeListRegionsRequest{}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeListRegions{LinodeListRegions: command},
	}
}

func (p *providerLinode) createListInstancesRequest() *protoapi.Request {
	command := &protoapi.LinodeListInstancesRequest{
		Auth: p.createAuth(),
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeListInstances{LinodeListInstances: command},
	}
}

func (p *providerLinode) createListImagesRequest() *protoapi.Request {
	command := &protoapi.LinodeListImagesRequest{
		Auth: p.createAuth(),
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeListImages{LinodeListImages: command},
	}
}

func (p *providerLinode) createListStackScripts() *protoapi.Request {
	command := &protoapi.LinodeListStackScriptsRequest{
		Auth: p.createAuth(),
	}
	return &protoapi.Request{
		R: &protoapi.Request_LinodeListStackscripts{LinodeListStackscripts: command},
	}
}

func (p *providerLinode) logInstance(instance *protoapi.LinodeInstance, msg string) {
	log.WithFields(log.Fields{
		"label":  instance.Label,
		"ipv4":   instance.Ipv4,
		"ipv6":   instance.Ipv6,
		"plan":   instance.Plan,
		"region": instance.Region,
		"status": strings.ToLower(instance.GetStatus().String()),
	}).Info(msg)
}

func (p *providerLinode) logError(msg string, errObject *protoapi.LinodeError, f ...log.Fields) {
	fields := log.Fields{}
	if len(f) > 0 {
		for k, v := range f[0] {
			fields[k] = v
	}
	}

	if hpErr := errObject.GetError(); hpErr != nil && len(hpErr.Message) > 0 {
		fields["server-error"] = hpErr.Message
		log.WithFields(fields).Error(msg)
	}

	if len(errObject.Details) == 1 {
		fields["cause"] = errObject.Details[0].Reason
		log.WithFields(fields).Error(msg)
	} else {
		for i, err := range errObject.GetDetails() {
			log.WithFields(log.Fields{
				"field":  err.Field,
				"reason": err.Reason,
			}).Errorf("Multiple errors. Error #%d", i)
		}
	}
}

func (p *providerLinode) parseDate(value string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05", value)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("2006-01-02T15:04:05-0700", value)
	if err == nil {
		return t, nil
	}
	return time.Unix(0, 0), err
}
