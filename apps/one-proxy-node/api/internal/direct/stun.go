package direct

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

const (
	stunBindingRequest      = 0x0001
	stunBindingSuccess      = 0x0101
	stunMappedAddress       = 0x0001
	stunXORMappedAddress    = 0x0020
	stunMagicCookie         = 0x2112A442
	defaultSTUNRoundTripTTL = 3 * time.Second
)

type CandidateGatherer struct {
	STUNServers []string
	Timeout     time.Duration
}

func (g CandidateGatherer) Gather(ctx context.Context, packetIO PacketIO) ([]domain.DirectCandidate, error) {
	candidates := gatherHostCandidates(packetIO)
	for _, server := range g.STUNServers {
		candidate, err := querySTUN(ctx, packetIO, server, g.Timeout)
		if err == nil {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("direct_candidates_not_found")
	}
	return candidates, nil
}

func gatherHostCandidates(packetIO PacketIO) []domain.DirectCandidate {
	port := localUDPPort(packetIO)
	candidates := make([]domain.DirectCandidate, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		return candidates
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := interfaceIP(addr)
			if ip == nil || ip.To4() == nil {
				continue
			}
			candidates = append(candidates, domain.DirectCandidate{
				Type:     domain.CandidateTypeHost,
				Address:  ip.String(),
				Port:     port,
				Protocol: domain.CandidateProtocolUDP,
				Priority: 100,
			})
		}
	}
	return candidates
}

func querySTUN(ctx context.Context, packetIO PacketIO, server string, timeout time.Duration) (domain.DirectCandidate, error) {
	if timeout <= 0 {
		timeout = defaultSTUNRoundTripTTL
	}
	addr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return domain.DirectCandidate{}, err
	}
	transactionID := make([]byte, 12)
	if _, err := rand.Read(transactionID); err != nil {
		return domain.DirectCandidate{}, err
	}
	request := make([]byte, 20)
	binary.BigEndian.PutUint16(request[0:2], stunBindingRequest)
	binary.BigEndian.PutUint32(request[4:8], stunMagicCookie)
	copy(request[8:20], transactionID)
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	queryCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if _, err := packetIO.WriteTo(request, addr); err != nil {
		return domain.DirectCandidate{}, err
	}
	buffer := make([]byte, 1500)
	for {
		n, _, err := packetIO.ReadNonQUICPacket(queryCtx, buffer)
		if err != nil {
			return domain.DirectCandidate{}, err
		}
		candidate, err := parseSTUNResponse(buffer[:n], transactionID, server)
		if err == nil {
			return candidate, nil
		}
	}
}

func parseSTUNResponse(packet []byte, transactionID []byte, server string) (domain.DirectCandidate, error) {
	if len(packet) < 20 {
		return domain.DirectCandidate{}, errors.New("short_stun_response")
	}
	if binary.BigEndian.Uint16(packet[0:2]) != stunBindingSuccess {
		return domain.DirectCandidate{}, errors.New("unexpected_stun_type")
	}
	length := int(binary.BigEndian.Uint16(packet[2:4]))
	if len(packet) < 20+length || binary.BigEndian.Uint32(packet[4:8]) != stunMagicCookie {
		return domain.DirectCandidate{}, errors.New("invalid_stun_header")
	}
	if string(packet[8:20]) != string(transactionID) {
		return domain.DirectCandidate{}, errors.New("stun_transaction_mismatch")
	}
	offset := 20
	for offset+4 <= 20+length {
		attrType := binary.BigEndian.Uint16(packet[offset : offset+2])
		attrLen := int(binary.BigEndian.Uint16(packet[offset+2 : offset+4]))
		valueStart := offset + 4
		valueEnd := valueStart + attrLen
		if valueEnd > len(packet) {
			return domain.DirectCandidate{}, errors.New("invalid_stun_attribute")
		}
		if attrType == stunXORMappedAddress || attrType == stunMappedAddress {
			return parseAddressAttribute(packet[valueStart:valueEnd], attrType == stunXORMappedAddress, server)
		}
		offset = valueEnd + ((4 - attrLen%4) % 4)
	}
	return domain.DirectCandidate{}, errors.New("mapped_address_not_found")
}

func parseAddressAttribute(value []byte, xor bool, server string) (domain.DirectCandidate, error) {
	if len(value) < 8 || value[1] != 0x01 {
		return domain.DirectCandidate{}, errors.New("unsupported_mapped_address")
	}
	port := int(binary.BigEndian.Uint16(value[2:4]))
	ip := append([]byte(nil), value[4:8]...)
	if xor {
		port = port ^ int(stunMagicCookie>>16)
		cookie := make([]byte, 4)
		binary.BigEndian.PutUint32(cookie, stunMagicCookie)
		for i := range ip {
			ip[i] ^= cookie[i]
		}
	}
	return domain.DirectCandidate{
		Type:       domain.CandidateTypeServerReflexive,
		Address:    net.IP(ip).String(),
		Port:       port,
		Protocol:   domain.CandidateProtocolUDP,
		STUNServer: server,
		Priority:   200,
	}, nil
}

func interfaceIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}

func localUDPPort(packetIO PacketIO) int {
	addr, ok := packetIO.LocalAddr().(*net.UDPAddr)
	if !ok {
		return 0
	}
	return addr.Port
}

func candidateAddress(candidate domain.DirectCandidate) string {
	return net.JoinHostPort(candidate.Address, strconv.Itoa(candidate.Port))
}
