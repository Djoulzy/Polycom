package main

import (
	"fmt"
	"github.com/benschw/srv-lb/lb"
	"github.com/miekg/dns"
	"strings"
)

func main() {
	zone := "$GENERATE 1-2 0 NS SERVER$.EXAMPLE.\n$GENERATE 1-8 $ CNAME $.0"
	to := dns.ParseZone(strings.NewReader(zone), "0.0.10.IN-ADDR.ARPA.", "")
	for x := range to {
		if x.Error == nil {
			fmt.Println(x.RR.String())
		}
	}

	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{"www.github.com/Djoulzy/Polycom.", dns.TypeSRV, dns.ClassINET}

	c := new(dns.Client)
	in, err := dns.Exchange(m1, "8.8.8.8:53")
	c.SingleInflight = true
	if t, ok := in.Answer[0].(*dns.TXT); ok {
		// do something with t.Txt
		fmt.Printf("%s\n", t.Txt)
	} else {
		fmt.Printf("%s\n", in)
	}

	srvName := "github.com/Djoulzy/Polycom"
	cfg, err := lb.DefaultConfig()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", cfg)
	l := lb.New(cfg, srvName)
	fmt.Printf("%s\n", l)

	address, err := l.Next()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", address.String())
	// Output: 0.1.2.3:8001
}
