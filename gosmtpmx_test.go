package gosmtpmx

import (
	"errors"
	"net"
	"reflect"
	"testing"
	"flag"
)

var integrate = flag.Bool("integrate", false, "Perform actual DNS lookup")

var testMX MX = MX{
	host: "",
	port: "25",
	auth: nil,
}

func NewTestClient(r Resolver, s Sender, mx MX) *client {
	return &client{
		Resolver: r,
		Sender:   s,
		mx:       mx,
	}
}

type testSender struct {
}

func (s testSender) SendMail(addr, from string, to []string, msg []byte) error {
	return nil
}

type testSenderRecoder struct {
	TestAddrs []string
	Errors    []error
	From      string
	To        []string
	Port      string
	Msg       []byte
}

func (s *testSenderRecoder) SendMail(addr, from string, to []string, msg []byte) error {
	s.TestAddrs = append(s.TestAddrs, addr)
	s.From = from
	s.To = to
	s.Msg = msg
	switch addr {
	case "127.0.0.3:10025":
		err := errors.New("[dummy] Connection refused")
		s.Errors = append(s.Errors, err)
		return err
	}
	return nil
}

type noSuchDomainResolver struct {
	defaultResolver
}

func (r noSuchDomainResolver) ResolvMX(host string) ([]*net.MX, error) {
	return nil, &net.DNSError{
		Name: host,
		Err:  "no such host",
	}
}

func (r noSuchDomainResolver) ResolvIP(host string) ([]net.IP, error) {
	return []net.IP{
		net.ParseIP("127.0.0.10"),
	}, nil
}

type testNormalResolver struct {
	defaultResolver
}

func (r testNormalResolver) ResolvMX(host string) ([]*net.MX, error) {
	switch host {
	case "example.org":
		return []*net.MX{
			&net.MX{Host: "mx1.example.org", Pref: 20},
			&net.MX{Host: "mx0.example.org", Pref: 10},
		}, nil
	case "example.net":
		return []*net.MX{
			&net.MX{Host: "mx1.example.org", Pref: 10},
			&net.MX{Host: "mx0.example.org", Pref: 20},
		}, nil
	case "example.info":
		return []*net.MX{
			&net.MX{Host: "mx0.example.info", Pref: 10},
			&net.MX{Host: "mx1.example.info", Pref: 20},
		}, nil
	}
	return []*net.MX{
		&net.MX{Host: "mx0.example.org", Pref: 20},
		&net.MX{Host: "nosuchmx0.example.org", Pref: 10},
	}, nil
}

func (r testNormalResolver) ResolvIP(host string) ([]net.IP, error) {
	switch host {
	case "mx0.example.org":
		return []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("127.0.0.2"),
		}, nil
	case "mx1.example.org":
		return []net.IP{
			net.ParseIP("127.0.0.3"),
			net.ParseIP("127.0.0.4"),
		}, nil
	case "mx0.example.info":
		return []net.IP{
			net.ParseIP("127.0.0.3"),
			net.ParseIP("127.0.0.3"),
		}, nil
	}
	return nil, errors.New("no such host")
}

type testFailureResolver struct {
	defaultResolver
}

func (r testFailureResolver) ResolvMX(host string) ([]*net.MX, error) {
	return nil, &net.DNSError{
		Name: host,
		Err:  "Unknown Error",
	}
}

func TestLookup_MX(t *testing.T) {
	c := NewTestClient(testNormalResolver{}, testSender{}, testMX)
	expected := map[uint16]*[]*net.MX{
		10: &[]*net.MX{
			&net.MX{
				Host: "mx0.example.org",
				Pref: 10,
			},
		},
		20: &[]*net.MX{
			&net.MX{
				Host: "mx1.example.org",
				Pref: 20,
			},
		},
	}
	list, err := c.LookupMX("example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != len(expected) {
		t.Fatalf("Expect lengh to be %d, but %d", len(expected), len(list))
	}
	if !reflect.DeepEqual(expected[0], list[0]) {
		t.Fatalf("Expect %v, but %v", expected, list)
	}
}

func TestLookupMX_ImplicitMX(t *testing.T) {
	c := NewTestClient(noSuchDomainResolver{}, testSender{}, testMX)
	expected := map[uint16]*[]*net.MX{
		0: &[]*net.MX{
			&net.MX{
				Host: "nosuchdomain.example.org",
				Pref: 0,
			},
		},
	}
	list, err := c.LookupMX("nosuchdomain.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != len(expected) {
		t.Fatal("Expect lengh to be %d, but %d", len(expected), len(list))
	}
	if !reflect.DeepEqual(expected[0], list[0]) {
		t.Fatalf("Expect %v, but %v", expected[0], list[0])
	}
}

func TestLookupMX_Failure(t *testing.T) {
	c := NewTestClient(testFailureResolver{}, testSender{}, testMX)
	list, err := c.LookupMX("example.org")
	if err == nil {
		t.Fatalf("Expect to be error, but nothing happen and returned '%v'", list)
	}
}

func TestLookupIP(t *testing.T) {
	c := NewTestClient(testNormalResolver{}, testSender{}, testMX)
	expected := []string{"127.0.0.1", "127.0.0.2"}
	ip, err := c.LookupIP("mx0.example.org")
	if err != nil {
		t.Fatalf("Expect not to be error, but '%v'", err)
	}
	if !reflect.DeepEqual(ip, expected) {
		t.Fatalf("Expect '%v', but '%v'", expected, ip)
	}
}

/*
Error situations
 1. LookupMX fails
   return immediately
 2. LookupIP fails
 3. SendMail fails
   try to next host if available, or try to next MX if available
   return error if no alternative found
*/

func TestDeliver_MultipleMX(t *testing.T) {
	mx := MX{
		host: "example.org",
		port: "10025",
	}
	s := &testSenderRecoder{}
	c := NewTestClient(testNormalResolver{}, s, mx)
	if err := c.Deliver("sender@example.net", []string{"rcpt1@example.com", "rcpt2@example.com"}, []byte("ABC")); err != nil {
		t.Fatalf("Expect not to be error, but '%v'", err)
	}

	if s.TestAddrs[0] != "127.0.0.1:10025" {
		t.Fatalf("Expect to try to connect to 1st preference, but connect to '%v'", s.TestAddrs[0])
	}
}

func TestDeliver_ImplicitMX(t *testing.T) {
	mx := MX{
		host: "example.org",
		port: "10025",
	}
	s := &testSenderRecoder{}
	c := NewTestClient(noSuchDomainResolver{}, s, mx)
	if err := c.Deliver("sender@example.net", []string{"rcpt1@example.com"}, []byte("ABC")); err != nil {
		t.Fatalf("Expect not to be error, but '%v'", err)
	}
	if s.TestAddrs[0] != "127.0.0.10:10025" {
		t.Fatalf("Expect to try to implicit MX, but connect to '%v'", s.TestAddrs[0])
	}
}

func TestDeliver_LookupIP(t *testing.T) {
	mx := MX{
		host: "nosuchmx.example.org",
		port: "10025",
	}
	s := &testSenderRecoder{}
	c := NewTestClient(testNormalResolver{}, s, mx)
	if err := c.Deliver("sender@example.net", []string{"rcpt1@example.com"}, []byte("ABC")); err != nil {
		t.Fatalf("Expect not to be error, but '%v'", err)
	}

	if s.TestAddrs[0] != "127.0.0.1:10025" {
		t.Fatalf("Expect to try to 2nd MX, but connect to '%v'", s.TestAddrs[0])
	}
}

func TestDeliver_SendMail(t *testing.T) {
	mx := MX{
		host: "example.net",
		port: "10025",
	}
	s := &testSenderRecoder{}
	c := NewTestClient(testNormalResolver{}, s, mx)
	if err := c.Deliver("sender@example.net", []string{"rcpt1@example.com"}, []byte("ABC")); err != nil {
		t.Fatalf("Expect not to be error, but '%v'", err)
	}

	if len(s.Errors) != 1 {
		t.Fatalf("Expect to be error, but no error found")
	}
	if s.Errors[0].Error() != "[dummy] Connection refused" {
		t.Fatalf("Expect to be connection refused, but '%v'", s.Errors[0])
	}
	if len(s.TestAddrs) != 2 {
		t.Fatalf("Expect to try 2 times, but '%v'", len(s.TestAddrs))
	}
	if s.TestAddrs[1] != "127.0.0.4:10025" {
		t.Fatalf("Expect to connect to 2nd host, but '%v'", s.TestAddrs[1])
	}
}

func TestDeliver_NoAlternative(t *testing.T) {
	mx := MX{
		host: "example.info",
		port: "10025",
	}
	s := &testSenderRecoder{}
	c := NewTestClient(testNormalResolver{}, s, mx)
	err := c.Deliver("sender@example.net", []string{"rcpt1@example.com"}, []byte("ABC"))
	if err == nil {
		t.Fatalf("Expect to be error, but no error found")
	}
	if err.Error() != "No alternative found" {
		t.Fatalf("Expect to be 'no alternative found', but '%v'", err)
	}

	expectedTestAddrs := []string{
		"127.0.0.3:10025",
		"127.0.0.3:10025",
	}
	if !reflect.DeepEqual(s.TestAddrs, expectedTestAddrs) {
		t.Fatalf("Expect to try to connect to 2 nodes, but '%v'", s.TestAddrs)
	}
}

func Test_New(t *testing.T) {
	if !*integrate {
		t.Skip("Actual DNS lookups are disabled. Add -integrate to perform lookups.")
	}

	mx := MX{
		host: "gosmtpmxtest.aws.tknetworks.org",
		port: "10025",
	}

	c := New(mx)
	{
		// all attempts failed
		err := c.Deliver("bad@example.org", []string{"rcpt1@example.com", "rcpt2@example.com"}, []byte("ABC"))
		if err.Error() != "No alternative found" {
			t.Fatal(err)
		}
	}
	/*
	{
		err := c.Deliver("good@example.org", []string{"rcpt1@example.com", "rcpt2@example.com"}, []byte("ABC"))
		if err != nil {
			t.Fatal(err)
		}
	}
	*/
}
