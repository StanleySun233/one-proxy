package direct

import (
	"context"
	"encoding/binary"
	"net"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func TestGatherSTUNCandidate(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go func() {
		buffer := make([]byte, 1500)
		n, addr, err := server.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		response := stunResponse(buffer[:n], net.IPv4(203, 0, 113, 7), 45123)
		_, _ = server.WriteToUDP(response, addr)
	}()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	gatherer := CandidateGatherer{STUNServers: []string{server.LocalAddr().String()}}
	candidates, err := gatherer.Gather(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if candidate.Type == domain.CandidateTypeServerReflexive && candidate.Address == "203.0.113.7" && candidate.Port == 45123 {
			return
		}
	}
	t.Fatalf("srflx candidate not found: %#v", candidates)
}

func TestClassifyNAT(t *testing.T) {
	endpointIndependent := ClassifyNAT([]domain.DirectCandidate{
		{Type: domain.CandidateTypeServerReflexive, Address: "203.0.113.7", Port: 45123},
		{Type: domain.CandidateTypeServerReflexive, Address: "203.0.113.7", Port: 45123},
	})
	if endpointIndependent.NATType != domain.NATTypeEndpointIndependent {
		t.Fatalf("want endpoint independent, got %s", endpointIndependent.NATType)
	}
	addressDependent := ClassifyNAT([]domain.DirectCandidate{
		{Type: domain.CandidateTypeServerReflexive, Address: "203.0.113.7", Port: 45123},
		{Type: domain.CandidateTypeServerReflexive, Address: "203.0.113.7", Port: 45124},
	})
	if addressDependent.NATType != domain.NATTypeAddressDependent {
		t.Fatalf("want address dependent, got %s", addressDependent.NATType)
	}
	blocked := ClassifyNAT(nil)
	if blocked.NATType != domain.NATTypeBlocked {
		t.Fatalf("want blocked, got %s", blocked.NATType)
	}
}

func stunResponse(request []byte, ip net.IP, port int) []byte {
	response := make([]byte, 32)
	binary.BigEndian.PutUint16(response[0:2], stunBindingSuccess)
	binary.BigEndian.PutUint16(response[2:4], 12)
	binary.BigEndian.PutUint32(response[4:8], stunMagicCookie)
	copy(response[8:20], request[8:20])
	binary.BigEndian.PutUint16(response[20:22], stunXORMappedAddress)
	binary.BigEndian.PutUint16(response[22:24], 8)
	response[25] = 0x01
	binary.BigEndian.PutUint16(response[26:28], uint16(port^int(stunMagicCookie>>16)))
	cookie := make([]byte, 4)
	binary.BigEndian.PutUint32(cookie, stunMagicCookie)
	ip4 := ip.To4()
	for i := range ip4 {
		response[28+i] = ip4[i] ^ cookie[i]
	}
	return response
}
