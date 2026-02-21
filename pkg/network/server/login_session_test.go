package server

import (
	"net"
	"testing"
	"time"

	"github.com/lulaide/gomc/pkg/network/crypt"
	"github.com/lulaide/gomc/pkg/network/protocol"
)

func TestLoginSessionOfflineHandshake(t *testing.T) {
	s := NewStatusServer(StatusConfig{
		ListenAddress: ":0",
		MOTD:          "GoMC",
		MaxPlayers:    20,
		VersionName:   "1.6.4",
		OnlineMode:    false,
	})

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		s.handleConn(serverConn)
		close(done)
	}()

	if err := protocol.WritePacket(clientConn, &protocol.Packet2ClientProtocol{
		ProtocolVersion: protocol.ProtocolVersion,
		Username:        "Steve",
		ServerHost:      "localhost",
		ServerPort:      25565,
	}); err != nil {
		t.Fatalf("write Packet2 failed: %v", err)
	}

	resp, err := protocol.ReadPacket(clientConn, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("read Packet253 failed: %v", err)
	}
	auth, ok := resp.(*protocol.Packet253ServerAuthData)
	if !ok {
		t.Fatalf("expected Packet253ServerAuthData, got %T", resp)
	}
	if auth.ServerID != "-" {
		t.Fatalf("offline server id mismatch: got=%q want=-", auth.ServerID)
	}

	pub, err := crypt.DecodePublicKey(auth.PublicKey)
	if err != nil {
		t.Fatalf("decode public key failed: %v", err)
	}

	sharedKey := []byte("0123456789abcdef")
	encKey, err := crypt.EncryptData(pub, sharedKey)
	if err != nil {
		t.Fatalf("encrypt shared key failed: %v", err)
	}
	encToken, err := crypt.EncryptData(pub, auth.VerifyToken)
	if err != nil {
		t.Fatalf("encrypt verify token failed: %v", err)
	}

	if err := protocol.WritePacket(clientConn, &protocol.Packet252SharedKey{
		SharedSecret: encKey,
		VerifyToken:  encToken,
	}); err != nil {
		t.Fatalf("write Packet252 failed: %v", err)
	}

	// Client switches output encryption immediately after sending Packet252.
	clientEncryptedOut, err := crypt.EncryptOutputStream(sharedKey, clientConn)
	if err != nil {
		t.Fatalf("encrypt output stream failed: %v", err)
	}

	ack, err := protocol.ReadPacket(clientConn, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("read Packet252 ack failed: %v", err)
	}
	if _, ok := ack.(*protocol.Packet252SharedKey); !ok {
		t.Fatalf("expected Packet252SharedKey ack, got %T", ack)
	}

	// Client switches input decryption after receiving Packet252.
	clientDecryptedIn, err := crypt.DecryptInputStream(sharedKey, clientConn)
	if err != nil {
		t.Fatalf("decrypt input stream failed: %v", err)
	}

	writeErrCh := make(chan error, 1)
	go func() {
		writeErrCh <- protocol.WritePacket(clientEncryptedOut, &protocol.Packet205ClientCommand{ForceRespawn: 0})
	}()

	wantOrder := []uint8{250, 1, 6, 202, 8, 43, 16, 104, 13, 51, 4}
	for _, wantID := range wantOrder {
		packet, err := protocol.ReadPacket(clientDecryptedIn, protocol.DirectionClientbound)
		if err != nil {
			t.Fatalf("read login-init packet failed: %v", err)
		}
		if packet.PacketID() != wantID {
			t.Fatalf("packet order mismatch: got id=%d want id=%d", packet.PacketID(), wantID)
		}
	}
	if err := <-writeErrCh; err != nil {
		t.Fatalf("write Packet205 failed: %v", err)
	}

	_ = clientConn.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server session did not terminate after client close")
	}
}

func TestLoginSessionRejectsInvalidUsername(t *testing.T) {
	s := NewStatusServer(StatusConfig{
		ListenAddress: ":0",
		MOTD:          "GoMC",
		MaxPlayers:    20,
		VersionName:   "1.6.4",
		OnlineMode:    false,
	})

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	go s.handleConn(serverConn)

	if err := protocol.WritePacket(clientConn, &protocol.Packet2ClientProtocol{
		ProtocolVersion: protocol.ProtocolVersion,
		Username:        "ab\u00A7c",
		ServerHost:      "localhost",
		ServerPort:      25565,
	}); err != nil {
		t.Fatalf("write Packet2 failed: %v", err)
	}

	resp, err := protocol.ReadPacket(clientConn, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("read kick failed: %v", err)
	}
	kick, ok := resp.(*protocol.Packet255KickDisconnect)
	if !ok {
		t.Fatalf("expected Packet255KickDisconnect, got %T", resp)
	}
	if kick.Reason != "Invalid username!" {
		t.Fatalf("kick reason mismatch: got=%q want=%q", kick.Reason, "Invalid username!")
	}
}
