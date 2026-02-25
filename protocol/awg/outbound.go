package awg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/netip"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/transport/awg"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
	"go4.org/netipx"
)

var _ adapter.Outbound = (*Outbound)(nil)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.LegacyAwgOutboundOptions](registry, constant.TypeAwg, NewOutbound)
}

type Outbound struct {
	outbound.Adapter
	dev       *awg.Device
	address   []netip.Prefix
	dnsRouter adapter.DNSRouter
	logger    logger.ContextLogger
	started   bool
	ctx       context.Context
}

type awgPeer struct {
	Endpoint                    string
	PublicKey                   string
	PresharedKey                string
	AllowedIPs                  []netip.Prefix
	PersistentKeepaliveInterval uint16
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.LegacyAwgOutboundOptions) (adapter.Outbound, error) {
	if options.MTU == 0 {
		options.MTU = 1408
	}

	dial, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain() || common.Any(options.Peers, func(it option.AwgPeerOptions) bool {
			return M.ParseSocksaddr(it.Address).IsFqdn()
		}),
		ResolverOnDetour: true,
	})
	if err != nil {
		return nil, err
	}

	var allowedPrefixBuilder netipx.IPSetBuilder
	var excludedPrefixBuilder netipx.IPSetBuilder

	actualPeers := common.Map(options.Peers, func(it option.AwgPeerOptions) awgPeer {
		endpoint := it.Address
		if it.Port != 0 {
			endpoint = net.JoinHostPort(it.Address, format.ToString(it.Port))
		}
		return awgPeer{
			Endpoint:                    endpoint,
			PublicKey:                   it.PublicKey,
			PresharedKey:                it.PresharedKey,
			AllowedIPs:                  it.AllowedIPs,
			PersistentKeepaliveInterval: it.PersistentKeepaliveInterval,
		}
	})
	if len(actualPeers) == 0 {
		actualPeers = []awgPeer{{
			Endpoint:     options.ServerOptions.Build().String(),
			PublicKey:    options.PeerPublicKey,
			PresharedKey: options.PreSharedKey,
			AllowedIPs:   []netip.Prefix{netip.PrefixFrom(netip.IPv4Unspecified(), 0), netip.PrefixFrom(netip.IPv6Unspecified(), 0)},
		}}
	}

	for _, peer := range actualPeers {
		for _, prefix := range peer.AllowedIPs {
			allowedPrefixBuilder.AddPrefix(prefix)
		}

		host, _, err := net.SplitHostPort(peer.Endpoint)
		if err == nil {
			if addr, err := netip.ParseAddr(host); err == nil {
				excludedPrefixBuilder.Add(addr)
			}
		} else if addr, err := netip.ParseAddr(peer.Endpoint); err == nil {
			excludedPrefixBuilder.Add(addr)
		}
	}
	allowedIps, err := allowedPrefixBuilder.IPSet()
	if err != nil {
		return nil, err
	}
	excludedIps, err := excludedPrefixBuilder.IPSet()
	if err != nil {
		return nil, err
	}

	ipc, err := genIpcConfigOutbound(options, actualPeers)
	if err != nil {
		return nil, err
	}

	dev, err := awg.NewDevice(ctx, logger, dial, ipc, awg.DeviceOpts{
		UseIntegratedTun: !options.SystemInterface,
		Address:          options.LocalAddress,
		AllowedIps:       allowedIps.Prefixes(),
		ExcludedIps:      excludedIps.Prefixes(),
		MTU:              options.MTU,
	})
	if err != nil {
		return nil, err
	}

	return &Outbound{
		Adapter:   outbound.NewAdapterWithDialerOptions(constant.TypeAwg, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		dev:       dev,
		address:   options.LocalAddress,
		dnsRouter: service.FromContext[adapter.DNSRouter](ctx),
		logger:    logger,
		ctx:       ctx,
	}, nil
}

func genIpcConfigOutbound(opts option.LegacyAwgOutboundOptions, peers []awgPeer) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(opts.PrivateKey)
	if err != nil {
		return "", err
	}
	s := "private_key=" + hex.EncodeToString(privateKeyBytes)
	if opts.Jc != 0 {
		s += "\njc=" + format.ToString(opts.Jc)
	}
	if opts.Jmin != 0 {
		s += "\njmin=" + format.ToString(opts.Jmin)
	}
	if opts.Jmax != 0 {
		s += "\njmax=" + format.ToString(opts.Jmax)
	}
	if opts.S1 != 0 {
		s += "\ns1=" + format.ToString(opts.S1)
	}
	if opts.S2 != 0 {
		s += "\ns2=" + format.ToString(opts.S2)
	}
	if opts.S3 != 0 {
		s += "\ns3=" + format.ToString(opts.S3)
	}
	if opts.S4 != 0 {
		s += "\ns4=" + format.ToString(opts.S4)
	}
	if opts.H1 != "" {
		s += "\nh1=" + opts.H1
	}
	if opts.H2 != "" {
		s += "\nh2=" + opts.H2
	}
	if opts.H3 != "" {
		s += "\nh3=" + opts.H3
	}
	if opts.H4 != "" {
		s += "\nh4=" + opts.H4
	}
	if opts.I1 != "" {
		s += "\ni1=" + opts.I1
	}
	if opts.I2 != "" {
		s += "\ni2=" + opts.I2
	}
	if opts.I3 != "" {
		s += "\ni3=" + opts.I3
	}
	if opts.I4 != "" {
		s += "\ni4=" + opts.I4
	}
	if opts.I5 != "" {
		s += "\ni5=" + opts.I5
	}

	for _, peer := range peers {
		publicKeyBytes, err := base64.StdEncoding.DecodeString(peer.PublicKey)
		if err != nil {
			return "", err
		}
		s += "\npublic_key=" + hex.EncodeToString(publicKeyBytes)
		if peer.PresharedKey != "" {
			presharedKeyBytes, err := base64.StdEncoding.DecodeString(peer.PresharedKey)
			if err != nil {
				return "", err
			}
			s += "\npreshared_key=" + hex.EncodeToString(presharedKeyBytes)
		}
		if peer.Endpoint != "" {
			s += "\nendpoint=" + peer.Endpoint
		}
		if peer.PersistentKeepaliveInterval != 0 {
			s += "\npersistent_keepalive_interval=" + format.ToString(peer.PersistentKeepaliveInterval)
		}
		for _, allowedIp := range peer.AllowedIPs {
			s += "\nallowed_ip=" + allowedIp.String()
		}
	}
	return s, nil
}

func (o *Outbound) Start(stage adapter.StartStage) error {
	if err := o.dev.Start(stage); err != nil {
		return err
	}
	if stage == adapter.StartStatePostStart {
		go o.readyChecker()
	}
	return nil
}

func (o *Outbound) Close() error {
	return o.dev.Close()
}

func (o *Outbound) readyChecker() {
	defer func() {
		o.started = true
		monitoring.Get(o.ctx).TestNow(o.Tag())
	}()
	for i := 0; i < 10; i++ {
		if o.IsReady() {
			return
		}
		select {
		case <-o.ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (o *Outbound) IsReady() bool {
	return o.started
}

func (o *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	o.logger.InfoContext(ctx, "outbound connection to ", destination)
	return o.dev.DialContext(ctx, network, destination)
}

func (o *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	o.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	return o.dev.ListenPacket(ctx, destination)
}
