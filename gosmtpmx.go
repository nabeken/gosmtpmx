package gosmtpmx

import (
	"errors"
	"math/rand"
	"net"
	"net/smtp"
	"sort"
	"time"
)

func SendMailVia(mx string, a smtp.Auth, from string, to []string, msg []byte) error {
	list, err := LookupMX(mx)
	if err != nil {
		return err
	}
	list.Shuffle()
	for _, pref := range list.Keys() {
		for _, rr := range *list[pref] {
			addrs, err := net.LookupIP(rr.Host)
			if err != nil {
				return err
			}
			for _, addr := range addrs {
				serr := smtp.SendMail(addr.String(), a, from, to, msg)
				if serr != nil {
					continue
				}
				return nil
			}
		}
	}
	return errors.New("No alternative found")
}

type MXlist map[uint16]*[]*net.MX

// NewMXList return MXlist that groups by preference.
func NewMXList(rrs []*net.MX) MXlist {
	list := make(MXlist)
	for _, rr := range rrs {
		if _, ok := list[rr.Pref]; !ok {
			list[rr.Pref] = &[]*net.MX{rr}
			continue
		}
		*list[rr.Pref] = append(*list[rr.Pref], rr)
	}
	return list
}

// Shuffle shuffles MX RR between same preference in MXlist
func (list *MXlist) Shuffle() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// stretch...
	r.Perm(1000)
	for pref := range *list {
		rrs := (*list)[pref]
		for i, p := range r.Perm(len(*rrs)) {
			cur := (*rrs)[i]
			(*rrs)[i] = (*rrs)[p]
			(*rrs)[p] = cur
		}
	}
}

// Keys returns keys sorted by preference
func (list *MXlist) Keys() []uint16 {
	prefs := sort.IntSlice{}
	for pref := range *list {
		prefs = append(prefs, int(pref))
	}
	prefs.Sort()

	ret := []uint16{}
	for _, pref := range prefs {
		ret = append(ret, uint16(pref))
	}
	return ret
}

// LookupMX returns MXlist.
// If these is no MX RR, LookupMX uses 'implicit MX' RR instead.
func LookupMX(host string) (MXlist, error) {
	rrs, err := lookupMX(host)
	if err != nil {
		return nil, err
	}
	list := NewMXList(rrs)
	return list, nil
}

func lookupMX(host string) ([]*net.MX, error) {
	// net.LookupMX handls CNAME properly
	mx, err := net.LookupMX(host)
	if err != nil {
		if derr, ok := err.(*net.DNSError); ok && dnsNoSuchHost(derr) {
			// RFC5321 5.1. Locating the Target Host
			// fallback to implicit MX if no such host
			mx = []*net.MX{
				&net.MX{
					Host: derr.Name,
					Pref: 0,
				},
			}
			return mx, nil
		}
		return nil, err
	}
	return mx, nil
}

func dnsNoSuchHost(err *net.DNSError) bool {
	return err.Err == "no such host"
}
