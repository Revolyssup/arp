package provider

import "github.com/Revolyssup/arp/pkg/config"

type Provider interface {
	Provide(chan<- config.Dynamic)
}
