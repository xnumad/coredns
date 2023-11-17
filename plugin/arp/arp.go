package arp

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"os/exec"
	"strings"
)

var log = clog.NewWithPlugin("arp")

type Arp struct {
	Next plugin.Handler
	*Arpfile
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
	mac, err := ipToMac(ipAddr)
	hostname := mac //TODO: MAC to name via ethers file
	hostnames := a.LookupStaticAddr(mac)
	if len(hostnames) != 0 {
		hostname = hostnames[0]
	}
	if err != nil {
		return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
	}
	answers = a.ptr(qname, 3600, []string{hostname})

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

func ipToMac(ipAddr string) (macAddr string, e error) {
	out, err := exec.Command("ndisc6", "-q", ipAddr, "wlp2s0" /*TODO don't hardcode interface*/).Output()

	//TODO IPv4 addresses
	//arping doesn't have easy output, use some Go library

	if err != nil {
		return "", err
	}
	outStr := string(out)
	outStrStripped := strings.TrimSuffix(outStr, "\n")
	return outStrStripped, nil
}

func (a Arp) Name() string { return "arp" }

// from hosts.go
func (a *Arp) ptr(zone string, ttl uint32, names []string) []dns.RR {
	answers := make([]dns.RR, len(names))
	for i, n := range names {
		r := new(dns.PTR)
		r.Hdr = dns.RR_Header{Name: zone, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl}
		r.Ptr = dns.Fqdn(n)
		answers[i] = r
	}
	return answers
}
