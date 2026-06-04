package direct

import (
	"context"
	"net"

	"github.com/quic-go/quic-go"
)

type PacketIO interface {
	LocalAddr() net.Addr
	WriteTo([]byte, net.Addr) (int, error)
	ReadNonQUICPacket(context.Context, []byte) (int, net.Addr, error)
}

type UDPConnPacketIO struct {
	Conn *net.UDPConn
}

func (io UDPConnPacketIO) LocalAddr() net.Addr {
	return io.Conn.LocalAddr()
}

func (io UDPConnPacketIO) WriteTo(payload []byte, addr net.Addr) (int, error) {
	return io.Conn.WriteTo(payload, addr)
}

func (io UDPConnPacketIO) ReadNonQUICPacket(ctx context.Context, buffer []byte) (int, net.Addr, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := io.Conn.SetReadDeadline(deadline); err != nil {
			return 0, nil, err
		}
	}
	return io.Conn.ReadFrom(buffer)
}

type QUICPacketIO struct {
	Transport *quic.Transport
}

func (io QUICPacketIO) LocalAddr() net.Addr {
	return io.Transport.Conn.LocalAddr()
}

func (io QUICPacketIO) WriteTo(payload []byte, addr net.Addr) (int, error) {
	return io.Transport.WriteTo(payload, addr)
}

func (io QUICPacketIO) ReadNonQUICPacket(ctx context.Context, buffer []byte) (int, net.Addr, error) {
	return io.Transport.ReadNonQUICPacket(ctx, buffer)
}
