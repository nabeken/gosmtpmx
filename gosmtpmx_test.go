package gosmtpmx

import (
	"net"
	"reflect"
	"testing"
)

func TestLookupMX_ImplicitMX(t *testing.T) {
	expected := map[uint16]*[]*net.MX{
		0: &[]*net.MX{
			&net.MX{
				Host: "nosuchdomain.example.org",
				Pref: 0,
			},
		},
	}
	list, err := LookupMX("nosuchdomain.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != len(expected) {
		t.Fatal("Expect lengh to be %d, but %d", len(expected), len(list))
	}
	if !reflect.DeepEqual(expected[0], list[0]) {
		t.Fatalf("Expect %v, but %v", expected, list)
	}
}

func TestKeys(t *testing.T) {
	list := NewMXList([]*net.MX{
		&net.MX{
			Host: "10.mx.example.com",
			Pref: 10,
		},
		&net.MX{
			Host: "0.mx.example.com",
			Pref: 0,
		},
		&net.MX{
			Host: "5.mx.example.com",
			Pref: 5,
		},
		&net.MX{
			Host: "10.mx.example.com",
			Pref: 10,
		},
	})
	expected := []uint16{0, 5, 10}
	if !reflect.DeepEqual(list.Keys(), expected) {
		t.Fatalf("Expect '%v', but '%v'", expected, list.Keys())
	}
}

func TestShuffle(t *testing.T) {
	list := NewMXList([]*net.MX{
		&net.MX{
			Host: "10-a.mx.example.com",
			Pref: 10,
		},
		&net.MX{
			Host: "10-c.mx.example.com",
			Pref: 10,
		},
		&net.MX{
			Host: "10-b.mx.example.com",
			Pref: 10,
		},
	})
	for i := 0; i < 10; i++ {
		list.Shuffle()
		// Check Host in the first value
		if (*list[10])[0].Host == "10-a.mx.example.com" {
			t.Log("Expect to be shuffled.. retrying..")
		} else {
			return
		}
	}
	t.Fatal("Expect to be shuffled")
}
