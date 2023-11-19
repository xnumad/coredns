package arp

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/mdlayher/arp"
	"github.com/miekg/dns"
	"net"
	"net/netip"
	"os/exec"
	"strings"
	"time"
)

var log = clog.NewWithPlugin("arp")

type Arp struct {
	Next plugin.Handler
	*Arpfile
	*arp.Client
	IfName string
}

func (a Arp) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()

	answers := []dns.RR{}

	if state.QType() != dns.TypePTR {
		return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
	}

	ipAddr := dnsutil.ExtractAddressFromReverse(qname)
	log.Debug("PTR for " + ipAddr)

	addr, err := netip.ParseAddr(ipAddr)
	if err != nil {
		return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
	}

	mac := ""
	if addr.Is6() {
		mac, err = a.ipv6ToMac(ipAddr)
		if err != nil {
			return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
		}
	} else if addr.Is4() {
		macTyped, err := a.ipv4ToMac(addr) //bug: doesn't work with interface's IPv4 addr (IPv6 works)
		if err != nil {
			return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
		}
		mac = macTyped.String()
	} else {
		return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
	}

	result := mac
	hostnames := a.LookupStaticAddr(mac)
	if len(hostnames) != 0 {
		result = hostnames[0]
	}
	answers = a.ptr(qname, 3600, []string{result})

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

func (a Arp) ipv4ToMac(addr netip.Addr) (macAddr net.HardwareAddr, e error) {
	if err := a.Client.SetDeadline(time.Now().Add(time.Second) /*absolute deadline*/); err != nil { //else DNS timeout
		return nil, err
	}

	hardwareAddr, err := a.Client.Resolve(addr)
	if err != nil {
		return nil, err
	}

	return hardwareAddr, nil
}

func (a Arp) ipv6ToMac(ipAddr string) (macAddr string, e error) {
	out, err := exec.Command("ndisc6", "-q", ipAddr, a.IfName).Output()

	if err != nil {
		//e.g. timeout
		return "", err
	}
	outStr := string(out)
	outStrStripped := strings.TrimSuffix(outStr, "\n")
	return outStrStripped, nil
}

func (a Arp) Name() string { return "arp" }

// from hosts.go
func (a Arp) ptr(zone string, ttl uint32, names []string) []dns.RR {
	answers := make([]dns.RR, len(names))
	for i, n := range names {
		r := new(dns.PTR)
		r.Hdr = dns.RR_Header{Name: zone, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl}
		r.Ptr = dns.Fqdn(n)
		answers[i] = r
	}
	return answers
}
