package proxycommand

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type Socks5Config struct {
	ListenAddr      string
	EntryHost       string
	EntryPort       int
	TokenEnv        string
	TokenFile       string
	PanelURL        string
	AccessTokenEnv  string
	AccessTokenFile string
	TenantID        string
	AccessPathID    string
	ConnectTimeout  time.Duration
}

func RunSocks5(cfg Socks5Config) error {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:1080"
	}
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go handleSocks5(conn, cfg)
	}
}

func handleSocks5(client net.Conn, cfg Socks5Config) {
	defer client.Close()
	targetHost, targetPort, err := socks5Handshake(client)
	if err != nil {
		return
	}
	upstream, err := DialUpstream(Config{
		EntryHost:       cfg.EntryHost,
		EntryPort:       cfg.EntryPort,
		TargetHost:      targetHost,
		TargetPort:      targetPort,
		TokenEnv:        cfg.TokenEnv,
		TokenFile:       cfg.TokenFile,
		PanelURL:        cfg.PanelURL,
		AccessTokenEnv:  cfg.AccessTokenEnv,
		AccessTokenFile: cfg.AccessTokenFile,
		TenantID:        cfg.TenantID,
		AccessPathID:    cfg.AccessPathID,
		ConnectTimeout:  cfg.ConnectTimeout,
	})
	if err != nil {
		_ = writeSocks5Reply(client, 0x05)
		return
	}
	defer upstream.Close()
	if err := writeSocks5Reply(client, 0x00); err != nil {
		return
	}
	bridge(client, upstream)
}

func socks5Handshake(conn net.Conn) (string, int, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", 0, err
	}
	if header[0] != 0x05 || header[1] == 0 {
		return "", 0, errors.New("invalid_socks5_greeting")
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", 0, err
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return "", 0, err
	}
	request := make([]byte, 4)
	if _, err := io.ReadFull(conn, request); err != nil {
		return "", 0, err
	}
	if request[0] != 0x05 || request[1] != 0x01 {
		return "", 0, errors.New("unsupported_socks5_command")
	}
	host, err := readSocks5Address(conn, request[3])
	if err != nil {
		return "", 0, err
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return "", 0, err
	}
	return host, int(binary.BigEndian.Uint16(portBytes)), nil
}

func readSocks5Address(conn net.Conn, atyp byte) (string, error) {
	switch atyp {
	case 0x01:
		raw := make([]byte, 4)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return "", err
		}
		return net.IP(raw).String(), nil
	case 0x03:
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return "", err
		}
		raw := make([]byte, int(length[0]))
		if _, err := io.ReadFull(conn, raw); err != nil {
			return "", err
		}
		return string(raw), nil
	case 0x04:
		raw := make([]byte, 16)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return "", err
		}
		return net.IP(raw).String(), nil
	default:
		return "", fmt.Errorf("unsupported_address_type_%d", atyp)
	}
}

func writeSocks5Reply(conn net.Conn, status byte) error {
	_, err := conn.Write([]byte{0x05, status, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	return err
}

func bridge(left net.Conn, right net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(right, left)
		if halfCloser, ok := right.(interface{ CloseWrite() error }); ok {
			_ = halfCloser.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(left, right)
		if halfCloser, ok := left.(interface{ CloseWrite() error }); ok {
			_ = halfCloser.CloseWrite()
		}
		done <- struct{}{}
	}()
	<-done
}
