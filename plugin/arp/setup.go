package arp

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register("arp", setup) }

func setup(c *caddy.Controller) error {
	h, err := arpParse(c)
	if err != nil {
		return plugin.Error("arp", err)
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		h.Next = next
		return h
	})

	return nil
}

func arpParse(c *caddy.Controller) (Arp, error) {
	h := Arp{
		Arpfile: &Arpfile{
			path: "/mnt/tmpfs/ethers", //TODO some other file
			hmap: newMap(),
		},
	}

	c.OnStartup(func() error {
		h.readEthers()
		return nil
	})

	return h, nil
}
