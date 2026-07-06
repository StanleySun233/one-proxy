package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

type guacamoleInstruction struct {
	Opcode string
	Args   []string
	Raw    string
}

type tcpAccessAuthFrame struct {
	Token        string   `json:"token"`
	TargetHost   string   `json:"targetHost"`
	TargetPort   int      `json:"targetPort"`
	ChainNodeIDs []string `json:"chainNodeIds,omitempty"`
}

type tcpAccessResponseFrame struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (c *ControlPlane) ServeRemoteTunnel(w http.ResponseWriter, req *http.Request, sessionID string, token string) {
	if sessionID == "" || token == "" {
		http.Error(w, "remote_session_required", http.StatusUnauthorized)
		return
	}
	record, ok := c.consumeRemoteSession(sessionID, token)
	if !ok {
		http.Error(w, "remote_session_invalid", http.StatusUnauthorized)
		return
	}
	listener, bridgeHost, bridgePort, err := startRemoteTCPBridge(record)
	if err != nil {
		http.Error(w, "remote_bridge_failed", http.StatusBadGateway)
		return
	}
	defer listener.Close()
	upgrader := websocket.Upgrader{Subprotocols: []string{"guacamole"}}
	wsConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	defer wsConn.Close()
	guacdConn, guacdReader, err := c.connectGuacd(record, bridgeHost, bridgePort)
	if err != nil {
		_ = wsConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "remote_connect_failed"), time.Now().Add(time.Second))
		return
	}
	defer guacdConn.Close()
	pipeGuacamole(wsConn, guacdConn, guacdReader)
}

func startRemoteTCPBridge(record remoteSessionRecord) (net.Listener, string, int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", 0, err
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, "", 0, errors.New("invalid_bridge_listener")
	}
	go acceptRemoteTCPBridge(listener, record)
	return listener, "127.0.0.1", addr.Port, nil
}

func acceptRemoteTCPBridge(listener net.Listener, record remoteSessionRecord) {
	clientConn, err := listener.Accept()
	if err != nil {
		return
	}
	defer clientConn.Close()
	upstreamConn, err := net.DialTimeout("tcp", net.JoinHostPort(record.TCPAccessHost, strconv.Itoa(record.TCPAccessPort)), 10*time.Second)
	if err != nil {
		return
	}
	defer upstreamConn.Close()
	if err := json.NewEncoder(upstreamConn).Encode(tcpAccessAuthFrame{
		Token:        record.ProxyToken,
		TargetHost:   record.TargetHost,
		TargetPort:   record.TargetPort,
		ChainNodeIDs: record.ChainNodeIDs,
	}); err != nil {
		return
	}
	upstreamReader := bufio.NewReader(upstreamConn)
	line, err := upstreamReader.ReadString('\n')
	if err != nil {
		return
	}
	var response tcpAccessResponseFrame
	if err := json.Unmarshal([]byte(line), &response); err != nil || response.Status != "connected" {
		return
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(upstreamConn, clientConn)
		_ = upstreamConn.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(clientConn, upstreamReader)
		_ = clientConn.Close()
	}()
	wg.Wait()
}

func (c *ControlPlane) connectGuacd(record remoteSessionRecord, bridgeHost string, bridgePort int) (net.Conn, *bufio.Reader, error) {
	guacdAddr := c.guacdAddr
	if guacdAddr == "" {
		guacdAddr = "127.0.0.1:4822"
	}
	conn, err := net.DialTimeout("tcp", guacdAddr, 10*time.Second)
	if err != nil {
		return nil, nil, err
	}
	reader := bufio.NewReader(conn)
	if err := writeGuacamoleInstruction(conn, "select", record.Protocol); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	args, err := readGuacamoleInstruction(reader)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if args.Opcode != "args" {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("unexpected_guacd_instruction:%s", args.Opcode)
	}
	if err := writeGuacamoleInstruction(conn, "size", strconv.Itoa(record.Width), strconv.Itoa(record.Height), strconv.Itoa(record.DPI)); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := writeGuacamoleInstruction(conn, "audio"); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := writeGuacamoleInstruction(conn, "video"); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := writeGuacamoleInstruction(conn, "image", "image/png", "image/jpeg"); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := writeGuacamoleInstruction(conn, "timezone", "UTC"); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	values := make([]string, 0, len(args.Args))
	for _, name := range args.Args {
		values = append(values, remoteGuacamoleArgument(record, name, bridgeHost, bridgePort))
	}
	if err := writeGuacamoleInstruction(conn, "connect", values...); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, reader, nil
}

func remoteGuacamoleArgument(record remoteSessionRecord, name string, bridgeHost string, bridgePort int) string {
	switch name {
	case "hostname":
		return bridgeHost
	case "port":
		return strconv.Itoa(bridgePort)
	case "username":
		return record.Username
	case "password":
		return record.Password
	case "private-key":
		return record.PrivateKey
	case "passphrase":
		return record.Passphrase
	case "width":
		return strconv.Itoa(record.Width)
	case "height":
		return strconv.Itoa(record.Height)
	case "dpi":
		return strconv.Itoa(record.DPI)
	case "ignore-cert":
		return "true"
	case "security":
		if record.Protocol == "rdp" {
			return "any"
		}
	case "color-depth":
		if record.Protocol == "rdp" {
			return "24"
		}
	case "resize-method":
		if record.Protocol == "rdp" {
			return "display-update"
		}
	}
	if strings.HasPrefix(name, "VERSION_") {
		return name
	}
	return ""
}

func pipeGuacamole(wsConn *websocket.Conn, guacdConn net.Conn, guacdReader *bufio.Reader) {
	done := make(chan struct{}, 2)
	var writeMu sync.Mutex
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			instruction, err := readGuacamoleInstruction(guacdReader)
			if err != nil {
				return
			}
			writeMu.Lock()
			err = wsConn.WriteMessage(websocket.TextMessage, []byte(instruction.Raw))
			writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			messageType, payload, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				continue
			}
			if isGuacamoleInternalPing(string(payload)) {
				writeMu.Lock()
				err = wsConn.WriteMessage(websocket.TextMessage, payload)
				writeMu.Unlock()
				if err != nil {
					return
				}
				continue
			}
			if _, err := guacdConn.Write(payload); err != nil {
				return
			}
		}
	}()
	<-done
	_ = guacdConn.Close()
	_ = wsConn.Close()
}

func isGuacamoleInternalPing(message string) bool {
	return strings.HasPrefix(message, "0.,4.ping,") && strings.HasSuffix(message, ";")
}

func writeGuacamoleInstruction(writer io.Writer, opcode string, args ...string) error {
	_, err := io.WriteString(writer, formatGuacamoleInstruction(opcode, args...))
	return err
}

func formatGuacamoleInstruction(opcode string, args ...string) string {
	elements := append([]string{opcode}, args...)
	var builder strings.Builder
	for index, element := range elements {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.Itoa(utf8.RuneCountInString(element)))
		builder.WriteByte('.')
		builder.WriteString(element)
	}
	builder.WriteByte(';')
	return builder.String()
}

func readGuacamoleInstruction(reader *bufio.Reader) (guacamoleInstruction, error) {
	elements := make([]string, 0, 8)
	var raw strings.Builder
	for {
		element, delimiter, rawElement, err := readGuacamoleElement(reader)
		if err != nil {
			return guacamoleInstruction{}, err
		}
		elements = append(elements, element)
		raw.WriteString(rawElement)
		if delimiter == ';' {
			break
		}
		if delimiter != ',' {
			return guacamoleInstruction{}, errors.New("invalid_guacamole_delimiter")
		}
	}
	if len(elements) == 0 {
		return guacamoleInstruction{}, errors.New("empty_guacamole_instruction")
	}
	return guacamoleInstruction{
		Opcode: elements[0],
		Args:   append([]string(nil), elements[1:]...),
		Raw:    raw.String(),
	}, nil
}

func readGuacamoleElement(reader *bufio.Reader) (string, rune, string, error) {
	var lengthBuilder strings.Builder
	var raw strings.Builder
	for {
		value, _, err := reader.ReadRune()
		if err != nil {
			return "", 0, "", err
		}
		raw.WriteRune(value)
		if value == '.' {
			break
		}
		if value < '0' || value > '9' {
			return "", 0, "", errors.New("invalid_guacamole_length")
		}
		lengthBuilder.WriteRune(value)
	}
	lengthText := lengthBuilder.String()
	if lengthText == "" {
		return "", 0, "", errors.New("invalid_guacamole_length")
	}
	length, err := strconv.Atoi(lengthText)
	if err != nil || length < 0 {
		return "", 0, "", errors.New("invalid_guacamole_length")
	}
	var valueBuilder strings.Builder
	for i := 0; i < length; i++ {
		value, _, err := reader.ReadRune()
		if err != nil {
			return "", 0, "", err
		}
		valueBuilder.WriteRune(value)
		raw.WriteRune(value)
	}
	delimiter, _, err := reader.ReadRune()
	if err != nil {
		return "", 0, "", err
	}
	raw.WriteRune(delimiter)
	return valueBuilder.String(), delimiter, raw.String(), nil
}
