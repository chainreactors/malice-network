package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// PeerIdentity holds all identity information extracted from the mTLS connection.
// It is computed once per RPC call and stored in the context for downstream handlers.
type PeerIdentity struct {
	CommonName  string // cert Subject.CommonName (typically the operator name)
	Fingerprint string // SHA-256 hex of DER cert bytes
	RemoteAddr  string // peer IP:port
	RemoteIP    string // peer IP only
	IsLoopback  bool   // whether RemoteIP is 127.0.0.1 or ::1
}

type peerIdentityKey struct{}

// extractPeerIdentity pulls all identity fields from the gRPC peer's TLS state.
// Returns a fully populated PeerIdentity or an error with proper gRPC status code.
func extractPeerIdentity(ctx context.Context) (*PeerIdentity, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no peer information in context")
	}
	if p.AuthInfo == nil {
		return nil, status.Error(codes.Unauthenticated, "no auth info in peer")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "peer auth info is not TLS")
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no peer certificates presented")
	}

	cert := tlsInfo.State.PeerCertificates[0]

	// Compute fingerprint from raw DER bytes
	hash := sha256.Sum256(cert.Raw)
	fp := hex.EncodeToString(hash[:])

	// Use CommonName from PeerCertificates (consistent with what was signed)
	cn := cert.Subject.CommonName

	// Remote address
	var remoteAddr, remoteIP string
	var isLoopback bool
	if p.Addr != nil {
		remoteAddr = p.Addr.String()
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			remoteIP = host
			parsed := net.ParseIP(host)
			isLoopback = parsed != nil && parsed.IsLoopback()
		}
	}

	return &PeerIdentity{
		CommonName:  cn,
		Fingerprint: fp,
		RemoteAddr:  remoteAddr,
		RemoteIP:    remoteIP,
		IsLoopback:  isLoopback,
	}, nil
}

// contextWithIdentity stores PeerIdentity in context for downstream handlers.
func contextWithIdentity(ctx context.Context, id *PeerIdentity) context.Context {
	return context.WithValue(ctx, peerIdentityKey{}, id)
}

