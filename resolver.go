package gosmtpmx

import (
	"net"
)

// Resolver implements DNS lookup functions
type Resolver interface {
	ResolvIP(host string) ([]net.IP, error)
	ResolvMX(host string) ([]*net.MX, error)
	NoSuchHost(err error) (bool, string)
}

type defaultResolver struct{}

func (r defaultResolver) NoSuchHost(err error) (bool, string) {
	if derr, ok := err.(*net.DNSError); ok && derr.Err == "no such host" {
		return true, derr.Name
	}
	return false, ""
}

// ResolvMX resolves a hostname to MX RRs.
func (r defaultResolver) ResolvMX(host string) ([]*net.MX, error) {
	return net.LookupMX(host)
}

// ResolvMX resolves a hostname to MX RRs.
func (r defaultResolver) ResolvIP(host string) ([]net.IP, error) {
	return net.LookupIP(host)
}
