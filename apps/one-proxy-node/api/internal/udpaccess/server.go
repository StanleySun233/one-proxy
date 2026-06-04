package udpaccess

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"strconv"
	"time"
)

const defaultIdleTimeout = 2 * time.Minute

type Authorizer interface {
	Validate(ctx context.Context, token string) bool
}

type Packet struct {
	Token      string `json:"token"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
	Data       []byte `json:"data"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    []byte `json:"data,omitempty"`
}

type Server struct {
	authorizer Authorizer
	timeout    time.Duration
}

func New(authorizer Authorizer) *Server {
	return &Server{authorizer: authorizer, timeout: defaultIdleTimeout}
}

func (s *Server) Serve(conn *net.UDPConn) error {
	buffer := make([]byte, 65535)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return err
		}
		payload := append([]byte(nil), buffer[:n]...)
		go s.handle(conn, clientAddr, payload)
	}
}

func (s *Server) handle(serverConn *net.UDPConn, clientAddr *net.UDPAddr, payload []byte) {
	var packet Packet
	if err := json.Unmarshal(payload, &packet); err != nil {
		_ = writeResponse(serverConn, clientAddr, Response{Status: "failed", Message: "invalid_packet"})
		return
	}
	if s.authorizer != nil && !s.authorizer.Validate(context.Background(), packet.Token) {
		_ = writeResponse(serverConn, clientAddr, Response{Status: "failed", Message: "auth_required"})
		return
	}
	if packet.TargetHost == "" || packet.TargetPort <= 0 || len(packet.Data) == 0 {
		_ = writeResponse(serverConn, clientAddr, Response{Status: "failed", Message: "invalid_target"})
		return
	}
	data, err := s.roundTrip(packet)
	if err != nil {
		_ = writeResponse(serverConn, clientAddr, Response{Status: "failed", Message: err.Error()})
		return
	}
	_ = writeResponse(serverConn, clientAddr, Response{Status: "ok", Data: data})
}

func (s *Server) roundTrip(packet Packet) ([]byte, error) {
	timeout := s.timeout
	if timeout <= 0 {
		timeout = defaultIdleTimeout
	}
	target, err := net.ResolveUDPAddr("udp", net.JoinHostPort(packet.TargetHost, strconv.Itoa(packet.TargetPort)))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, target)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	if _, err := conn.Write(packet.Data); err != nil {
		return nil, err
	}
	buffer := make([]byte, 65535)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("empty_udp_response")
	}
	return append([]byte(nil), buffer[:n]...), nil
}

func writeResponse(conn *net.UDPConn, addr *net.UDPAddr, response Response) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(payload, addr)
	return err
}

func ListenAndServe(addr string, server *Server) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Printf("udp-access resolve failed addr=%s err=%v", addr, err)
		return
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("udp-access listen failed addr=%s err=%v", addr, err)
		return
	}
	defer conn.Close()
	log.Printf("udp-access listening addr=%s", conn.LocalAddr().String())
	if err := server.Serve(conn); err != nil {
		log.Printf("udp-access stopped addr=%s err=%v", conn.LocalAddr().String(), err)
	}
}
