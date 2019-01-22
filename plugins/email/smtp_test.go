package email

import (
	"testing"
)

var SMTPConf = map[string]interface{}{
	LabelHost:     "smtp.mailtrap.io",
	LabelPort:     2525,
	LabelUsername: " 302cf7c45f9d5f",
	LabelPassword: "05b7d0d170f3a8",
	LabelFrom:     "admin@example.com"}

func TestNewSMTPSender(t *testing.T) {
	send, err := NewSMTPSender(SMTPConf)
	if err != nil {
		t.Error(err)
	} else if send == nil {
		t.Errorf("No error occurred but the sender is nil")
	}

	s, ok := send.(*SMTPSender)
	if !ok {
		t.Error("NewSMTPSender didn't return an *SMTPSender")
	}

	if s.d.Username != SMTPConf[LabelUsername].(string) {
		t.Errorf("Usernames don't match. Expected %s, got %s",
			SMTPConf[LabelUsername], s.d.Username)
	}
	if s.d.Password != SMTPConf[LabelPassword].(string) {
		t.Errorf("Passwords don't match. Expected %s, got %s",
			SMTPConf[LabelPassword], s.d.Password)
	}
	if s.d.Host != SMTPConf[LabelHost].(string) {
		t.Errorf("Hosts don't match. Expected %s, got %s",
			SMTPConf[LabelHost], s.d.Host)
	}
	if s.d.Port != SMTPConf[LabelPort].(int) {
		t.Errorf("Ports don't match. Expected %d, got %d",
			SMTPConf[LabelPort], s.d.Port)
	}
	if s.from != SMTPConf[LabelFrom].(string) {
		t.Errorf("From addresses don't match. Expected %s, got %s",
			SMTPConf[LabelFrom], s.from)
	}
}
