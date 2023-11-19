package arp

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/mdlayher/arp"
	"net"
)

func init() { plugin.Register("arp", setup) }

func setup(c *caddy.Controller) error {
	h, err := arpParse(c)
	if err != nil {
		return plugin.Error("arp", err)
	}

	ifName := "wlp2s0" /*TODO don't hardcode interface*/

	//set up client
	ifIndex, err := net.InterfaceByName(ifName)
	if err != nil {
		return nil
	}
	client, err := arp.Dial(ifIndex)
	if err != nil {
		return nil
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		h.Next = next
		h.Client = client
		h.IfName = ifName
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
