package gosmtpmx

import (
	"errors"
	"net/smtp"
)

type MX struct {
	host string
	port string
	auth smtp.Auth
}

type client struct {
	Resolver
	Sender
	mx MX
}

func New(mx MX) *client {
	return &client{
		Resolver: defaultResolver{},
		Sender:   NewSender(mx),
		mx:       mx,
	}
}

func (c client) Deliver(from string, to []string, msg []byte) error {
	list, err := c.LookupMX(c.mx.host)
	if err != nil {
		return err
	}
	list.Shuffle()
	for _, pref := range list.Prefs() {
		for _, rr := range *list[pref] {
			addrs, err := c.LookupIP(rr.Host)
			if err != nil {
				// try to next host if available
				continue
			}
			for _, addr := range addrs {
				serr := c.SendMail(addr+":"+c.mx.port, from, to, msg)
				if serr != nil {
					continue
				}
				// return immediately if successed
				return nil
			}
		}
	}
	return errors.New("No alternative found")
}

// LookupIP resolves a hostname to IP address.
func (c client) LookupIP(host string) ([]string, error) {
	_addrs, err := c.ResolvIP(host)
	if err != nil {
		return nil, err
	}
	addrs := []string{}
	for _, addr := range _addrs {
		addrs = append(addrs, addr.String())
	}
	return addrs, nil
}

func (c client) LookupMX(host string) (MXlist, error) {
	rrs, err := c.ResolvMX(host)
	if err != nil {
		if ok, name := c.NoSuchHost(err); ok {
			// RFC5321 5.1. Locating the Target Host
			// fallback to implicit MX if no such host
			return NewImplicitMXList(name), nil
		}
		return nil, err
	}
	return NewMXList(rrs), nil
}
