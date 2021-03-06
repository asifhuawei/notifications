package smtpd_test

import (
	"bitbucket.org/chrj/smtpd"
	"errors"
	"net/smtp"
	"strings"
)

func ExampleServer() {
	var server *smtpd.Server

	// No-op server. Accepts and discards
	server = &smtpd.Server{}
	server.ListenAndServe()

	// Relay server. Accepts only from single IP address and forwards using the Gmail smtp
	server = &smtpd.Server{

		Addr: "0.0.0.0:10025",

		HeloChecker: func(peer smtpd.Peer, name string) error {
			if !strings.HasPrefix(peer.Addr.String(), "42.42.42.42:") {
				return errors.New("Denied")
			}
			return nil
		},

		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {

			return smtp.SendMail(
				"smtp.gmail.com:587",
				smtp.PlainAuth(
					"",
					"username@gmail.com",
					"password",
					"smtp.gmail.com",
				),
				env.Sender,
				env.Recipients,
				env.Data,
			)

		},
	}

	server.ListenAndServe()
}
