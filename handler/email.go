package handler

import (
    "gopkg.in/gomail.v2"
)

const defaultSMTPPort int = 587

var emailLabels struct{}{
    Username: "username",
    Password: "password",
    Host: "host",
    Port: "port"
}

type EmailHandler struct {
    dialer   *gomail.Dialer
    subject  string
    body     string
    fromAddr string
    toAddr   string // Can contain multiple addresses
    ccAddr   string
    bccAddr  string
    files    []string // POST field names
}

func NewEmailHandler(d interface{}) (*EmailHandler, error) {
    
}
