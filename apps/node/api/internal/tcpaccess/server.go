package tcpaccess

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

const authTimeout = 10 * time.Second

type Authorizer interface {
	Validate(ctx context.Context, token string) bool
}

type StreamOpener interface {
	OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error)
}

type Server struct {
	authorizer Authorizer
	streams    StreamOpener
	dial       func(context.Context, string, int) (net.Conn, error)
}

type AuthFrame struct {
	Token               string   `json:"token"`
	TargetHost          string   `json:"targetHost"`
	TargetPort          int      `json:"targetPort"`
	NextNodeID          string   `json:"nextNodeId,omitempty"`
	RemainingHopNodeIDs []string `json:"remainingHopNodeIds,omitempty"`
	ChainNodeIDs        []string `json:"chainNodeIds,omitempty"`
}

type responseFrame struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func New(authorizer Authorizer, streams StreamOpener) *Server {
	return &Server{
		authorizer: authorizer,
		streams:    streams,
		dial:       dialTCP,
	}
}

func (s *Server) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(authTimeout))
	reader := bufio.NewReader(conn)
	frame, err := readAuthFrame(reader)
	if err != nil {
		writeResponse(conn, "failed", "invalid_auth_frame")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()
	if s.authorizer == nil || frame.Token == "" || !s.authorizer.Validate(ctx, frame.Token) {
		writeResponse(conn, "failed", "auth_required")
		return
	}
	targetConn, err := s.connect(ctx, frame)
	if err != nil {
		writeResponse(conn, "failed", err.Error())
		return
	}
	defer targetConn.Close()
	_ = conn.SetDeadline(time.Time{})
	if err := writeResponse(conn, "connected", ""); err != nil {
		return
	}
	bridge(conn, reader, targetConn)
}

func (s *Server) connect(ctx context.Context, frame AuthFrame) (net.Conn, error) {
	if frame.TargetHost == "" || frame.TargetPort <= 0 || frame.TargetPort > 65535 {
		return nil, errors.New("invalid_target")
	}
	nextNodeID, remaining := chain(frame)
	if nextNodeID != "" {
		if s.streams == nil {
			return nil, errors.New("stream_registry_unavailable")
		}
		conn, err := s.streams.OpenStream(nextNodeID, remaining, frame.TargetHost, frame.TargetPort)
		if err != nil {
			return nil, errors.New("connect_failed")
		}
		return conn, nil
	}
	conn, err := s.dial(ctx, frame.TargetHost, frame.TargetPort)
	if err != nil {
		return nil, errors.New("connect_failed")
	}
	return conn, nil
}

func chain(frame AuthFrame) (string, []string) {
	if frame.NextNodeID != "" {
		return frame.NextNodeID, append([]string(nil), frame.RemainingHopNodeIDs...)
	}
	if len(frame.ChainNodeIDs) == 0 {
		return "", nil
	}
	return frame.ChainNodeIDs[0], append([]string(nil), frame.ChainNodeIDs[1:]...)
}

func readAuthFrame(reader *bufio.Reader) (AuthFrame, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return AuthFrame{}, err
	}
	var frame AuthFrame
	if err := json.Unmarshal([]byte(line), &frame); err != nil {
		return AuthFrame{}, err
	}
	return frame, nil
}

func writeResponse(conn net.Conn, status string, message string) error {
	payload, err := json.Marshal(responseFrame{Status: status, Message: message})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(conn, "%s\n", payload)
	return err
}

func dialTCP(ctx context.Context, host string, port int) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
}

func bridge(clientConn net.Conn, clientReader *bufio.Reader, targetConn net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(targetConn, clientReader)
		_ = targetConn.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(clientConn, targetConn)
		_ = clientConn.Close()
	}()
	wg.Wait()
}

func ListenAndServe(addr string, server *Server) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("tcp-access listen failed addr=%s err=%v", addr, err)
		return
	}
	log.Printf("tcp-access listening addr=%s", listener.Addr().String())
	if err := server.Serve(listener); err != nil {
		log.Printf("tcp-access stopped addr=%s err=%v", listener.Addr().String(), err)
	}
}
