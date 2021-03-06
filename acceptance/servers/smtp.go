package servers

import (
	"net"
	"os"

	"bitbucket.org/chrj/smtpd"
)

type SMTP struct {
	listener   net.Listener
	server     *smtpd.Server
	Deliveries []smtpd.Envelope
}

func NewSMTP() *SMTP {
	return &SMTP{
		server: &smtpd.Server{
			Addr: "127.0.0.1:0",
		},
		Deliveries: make([]smtpd.Envelope, 0),
	}
}

func (s *SMTP) Boot() {
	var err error
	s.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	s.server.Handler = s.Handler

	go s.server.Serve(s.listener)

	addr := s.listener.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}
	os.Setenv("SMTP_HOST", host)
	os.Setenv("SMTP_PORT", port)
}

func (s *SMTP) Handler(peer smtpd.Peer, env smtpd.Envelope) error {
	s.Deliveries = append(s.Deliveries, env)
	return nil
}

func (s *SMTP) Reset() {
	s.Deliveries = []smtpd.Envelope{}
}

func (s *SMTP) Close() {
	s.listener.Close()
}
