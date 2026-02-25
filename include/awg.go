//go:build with_awg

package include

import (
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/protocol/awg"
)

func registerAwgEndpoint(registry *endpoint.Registry) {
	awg.RegisterEndpoint(registry)
}

func registerAwgOutbound(registry *outbound.Registry) {
	awg.RegisterOutbound(registry)
}
