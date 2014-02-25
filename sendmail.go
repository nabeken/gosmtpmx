package gosmtpmx

import (
	"net/smtp"
)

type Sender interface {
    SendMail(addr, from string, to []string, msg []byte) error
}

type sender struct {
	mx MX
}

func NewSender(mx MX) Sender {
	return sender{
		mx: mx,
	}
}

// SendMail sends a message to addr usin net.smtp
func (s sender) SendMail(addr, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, s.mx.auth, from, to, msg)
}
