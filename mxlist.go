package gosmtpmx

import (
	"math/rand"
	"net"
	"sort"
	"time"
)

type MXlist map[uint16]*[]*net.MX

// NewMXList returns MXlist that groups []*net.MX by preference.
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

func NewImplicitMXList(name string) MXlist {
	rrs := []*net.MX{
		&net.MX{
			Host: name,
			Pref: 0,
		},
	}
	return NewMXList(rrs)
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

// Prefs returns sorted preferences
func (list *MXlist) Prefs() []uint16 {
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
