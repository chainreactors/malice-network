//go:build mockimplant

package testsupport

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	iomtypes "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	implantparser "github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	serverlistener "github.com/chainreactors/malice-network/server/listener"
	serverrpc "github.com/chainreactors/malice-network/server/rpc"
	"gopkg.in/yaml.v3"
)

func StartInProcessListener(t testing.TB, h *ControlPlaneHarness, listenerName string) {
	t.Helper()

	authConfig := h.NewListenerClientConfig(t, listenerName)
	authBytes, err := yaml.Marshal(authConfig)
	if err != nil {
		t.Fatalf("marshal listener auth: %v", err)
	}

	authPath, err := h.WriteTempFile(listenerName+".auth", authBytes)
	if err != nil {
		t.Fatalf("write listener auth: %v", err)
	}

	cfg := &configs.ListenerConfig{
		Enable: true,
		Name:   listenerName,
		Auth:   authPath,
		IP:     "127.0.0.1",
	}
	if err := serverlistener.NewListener(authConfig, cfg, true); err != nil {
		t.Fatalf("start listener %s failed: %v", listenerName, err)
	}

	t.Cleanup(func() {
		if serverlistener.Listener != nil {
			_ = serverlistener.Listener.Close()
		}
	})
}

func StartPipeline(t testing.TB, pipeline *clientpb.Pipeline) {
	t.Helper()

	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	if _, err := (&serverrpc.Server{}).RegisterPipeline(context.Background(), pipeline); err != nil {
		t.Fatalf("register pipeline %s failed: %v", pipeline.Name, err)
	}
	if _, err := (&serverrpc.Server{}).StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Pipeline:   pipeline,
	}); err != nil {
		t.Fatalf("start pipeline %s failed: %v", pipeline.Name, err)
	}

	t.Cleanup(func() {
		_, _ = (&serverrpc.Server{}).StopPipeline(context.Background(), &clientpb.CtrlPipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Pipeline:   pipeline,
		})
	})
}

func ReserveTCPPort(t testing.TB) uint16 {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port failed: %v", err)
	}
	defer ln.Close()

	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

func EnableSelfSignedTLS(t testing.TB, pipeline *clientpb.Pipeline) {
	t.Helper()

	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	caCertPEM, caKeyPEM, err := certs.GenerateCACert("test-pipeline", nil)
	if err != nil {
		t.Fatalf("generate ca cert: %v", err)
	}

	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		t.Fatal("decode ca cert pem failed")
	}
	caCertX509, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatalf("parse ca cert: %v", err)
	}

	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		t.Fatal("decode ca key pem failed")
	}
	caPrivKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatalf("parse ca key: %v", err)
	}

	serverCertPEM, serverKeyPEM, err := certs.GenerateChildCert("127.0.0.1", false, caCertX509, caPrivKey)
	if err != nil {
		t.Fatalf("generate server cert: %v", err)
	}

	pipeline.Tls = &clientpb.TLS{
		Enable: true,
		Cert: &clientpb.Cert{
			Cert: string(serverCertPEM),
			Key:  string(serverKeyPEM),
		},
		Ca: &clientpb.Cert{
			Cert: string(caCertPEM),
			Key:  string(caKeyPEM),
		},
	}
}

type TCPPacketImplant struct {
	t         testing.TB
	RawID     uint32
	SessionID string
	parser    *implantparser.MessageParser
	rawConn   net.Conn
	crypto    *cryptostream.CryptoConn
	pending   []*implantpb.Spite
}

func NewTCPPacketImplant(t testing.TB, pipeline *clientpb.Pipeline) *TCPPacketImplant {
	t.Helper()

	if pipeline == nil || pipeline.GetTcp() == nil {
		t.Fatal("tcp pipeline is nil")
	}

	cryptos, err := configs.NewCrypto(pipeline.GetEncryption())
	if err != nil {
		t.Fatalf("build pipeline cryptor failed: %v", err)
	}
	if len(cryptos) == 0 {
		t.Fatal("pipeline has no encryption configured")
	}

	parser, err := implantparser.NewParser(pipeline.GetParser())
	if err != nil {
		t.Fatalf("build parser failed: %v", err)
	}

	address := fmt.Sprintf("%s:%d", pipeline.GetTcp().GetHost(), pipeline.GetTcp().GetPort())
	rawConn, err := dialPipelineConn(address, pipeline.GetTls())
	if err != nil {
		t.Fatalf("dial pipeline %s failed: %v", address, err)
	}

	rawID := uint32(time.Now().UnixNano())
	implant := &TCPPacketImplant{
		t:         t,
		RawID:     rawID,
		SessionID: hash.Md5Hash(encoders.Uint32ToBytes(rawID)),
		parser:    parser,
		rawConn:   rawConn,
		crypto:    cryptostream.NewCryptoConn(rawConn, cryptos[0]),
	}

	t.Cleanup(func() {
		_ = implant.rawConn.Close()
	})

	return implant
}

func (i *TCPPacketImplant) Send(spite *implantpb.Spite) error {
	if i == nil {
		return errors.New("tcp mock implant is nil")
	}
	if spite == nil {
		return errors.New("spite is nil")
	}
	return i.parser.WritePacket(i.crypto, iomtypes.BuildOneSpites(spite), i.RawID)
}

func (i *TCPPacketImplant) WritePlain(data []byte) error {
	if i == nil {
		return errors.New("tcp mock implant is nil")
	}
	if len(data) == 0 {
		return errors.New("data is empty")
	}
	n, err := i.crypto.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("short write: got %d, want %d", n, len(data))
	}
	return nil
}

func (i *TCPPacketImplant) Read(timeout time.Duration) (*implantpb.Spite, error) {
	if i == nil {
		return nil, errors.New("tcp mock implant is nil")
	}
	if len(i.pending) > 0 {
		spite := i.pending[0]
		i.pending = i.pending[1:]
		return spite, nil
	}
	if err := i.rawConn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer func() {
		_ = i.rawConn.SetReadDeadline(time.Time{})
	}()

	rawID, spites, err := i.parser.ReadPacket(i.crypto)
	if err != nil {
		return nil, err
	}
	if rawID != i.RawID {
		return nil, fmt.Errorf("server packet raw id = %d, want %d", rawID, i.RawID)
	}
	if spites == nil || len(spites.Spites) == 0 {
		return nil, errors.New("server packet contained no spites")
	}
	if len(spites.Spites) > 1 {
		i.pending = append(i.pending, spites.Spites[1:]...)
	}
	return spites.Spites[0], nil
}

func WaitForTCPModuleRequest(t testing.TB, implant *TCPPacketImplant, module string, timeout time.Duration) *implantpb.Spite {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := implant.Send(iomtypes.BuildPingSpite()); err != nil {
			t.Fatalf("send ping trigger failed: %v", err)
		}

		spite, err := implant.Read(1500 * time.Millisecond)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			t.Fatalf("read server packet failed: %v", err)
		}

		if spite.GetInit() != nil || spite.GetPing() != nil {
			continue
		}
		if spite.GetName() == module {
			return spite
		}
		t.Fatalf("server request = %q, want %q", spite.GetName(), module)
	}

	t.Fatalf("timed out waiting for server request %s", module)
	return nil
}

func DrainTCPServerPackets(t testing.TB, implant *TCPPacketImplant, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		spite, err := implant.Read(50 * time.Millisecond)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return
			}
			t.Fatalf("drain server packet failed: %v", err)
		}
		if spite == nil {
			return
		}
	}
}

type HTTPPacketImplant struct {
	t           testing.TB
	RawID       uint32
	SessionID   string
	parser      *implantparser.MessageParser
	endpoint    string
	encryptions []*clientpb.Encryption
	bodyPrefix  []byte
	bodySuffix  []byte
	client      *http.Client
}

func NewHTTPPacketImplant(t testing.TB, pipeline *clientpb.Pipeline) *HTTPPacketImplant {
	t.Helper()

	if pipeline == nil || pipeline.GetHttp() == nil {
		t.Fatal("http pipeline is nil")
	}

	params, err := implanttypes.UnmarshalPipelineParams(pipeline.GetHttp().GetParams())
	if err != nil {
		t.Fatalf("unmarshal http params failed: %v", err)
	}

	parser, err := implantparser.NewParser(pipeline.GetParser())
	if err != nil {
		t.Fatalf("build parser failed: %v", err)
	}

	scheme := "http"
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if pipeline.GetTls().GetEnable() {
		clientTLSConfig, err := clientTLSConfig(pipeline.GetTls())
		if err != nil {
			t.Fatalf("build https client tls config failed: %v", err)
		}
		transport.TLSClientConfig = clientTLSConfig
		scheme = "https"
	}

	rawID := uint32(time.Now().UnixNano())
	return &HTTPPacketImplant{
		t:           t,
		RawID:       rawID,
		SessionID:   hash.Md5Hash(encoders.Uint32ToBytes(rawID)),
		parser:      parser,
		endpoint:    fmt.Sprintf("%s://%s:%d/", scheme, pipeline.GetHttp().GetHost(), pipeline.GetHttp().GetPort()),
		encryptions: pipeline.GetEncryption(),
		bodyPrefix:  []byte(params.BodyPrefix),
		bodySuffix:  []byte(params.BodySuffix),
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}
}

func (i *HTTPPacketImplant) Exchange(spite *implantpb.Spite) ([]*implantpb.Spite, *http.Response, error) {
	if i == nil {
		return nil, nil, errors.New("http mock implant is nil")
	}
	if spite == nil {
		return nil, nil, errors.New("spite is nil")
	}

	packet, err := i.parser.Marshal(iomtypes.BuildOneSpites(spite), i.RawID)
	if err != nil {
		return nil, nil, err
	}
	return i.ExchangePlain(packet)
}

func (i *HTTPPacketImplant) ExchangePlain(packet []byte) ([]*implantpb.Spite, *http.Response, error) {
	if i == nil {
		return nil, nil, errors.New("http mock implant is nil")
	}
	if len(packet) == 0 {
		return nil, nil, errors.New("packet is empty")
	}

	cryptos, err := requestCryptos(i.encryptions)
	if err != nil {
		return nil, nil, err
	}

	requestBody, err := cryptostream.Encrypt(cryptos[0], packet)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, i.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, resp, err
	}

	body, err = trimHTTPBodyEnvelope(body, i.bodyPrefix, i.bodySuffix)
	if err != nil {
		return nil, resp, err
	}
	if len(body) == 0 {
		return nil, resp, nil
	}

	plainResponse, err := cryptostream.Decrypt(cryptos[0], body)
	if err != nil {
		return nil, resp, err
	}

	responseConn := cryptostream.WrapReadWriteCloser(bytes.NewReader(plainResponse), io.Discard, nil)
	rawID, spites, err := i.parser.ReadPacket(responseConn)
	if err != nil {
		return nil, resp, err
	}
	if rawID != i.RawID {
		return nil, resp, fmt.Errorf("http response raw id = %d, want %d", rawID, i.RawID)
	}
	if spites == nil {
		return nil, resp, nil
	}
	return spites.Spites, resp, nil
}

func WaitForHTTPModuleRequest(implant *HTTPPacketImplant, module string, timeout time.Duration) (*implantpb.Spite, *http.Response, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		spites, resp, err := implant.Exchange(iomtypes.BuildPingSpite())
		if err != nil {
			return nil, nil, err
		}
		if spite := findSpiteByName(spites, module); spite != nil {
			return spite, resp, nil
		}
	}
	return nil, nil, fmt.Errorf("timed out waiting for http request %s", module)
}

func dialPipelineConn(address string, tlsConfigPB *clientpb.TLS) (net.Conn, error) {
	if tlsConfigPB == nil || !tlsConfigPB.GetEnable() {
		return net.DialTimeout("tcp", address, 5*time.Second)
	}

	cfg, err := clientTLSConfig(tlsConfigPB)
	if err != nil {
		return nil, err
	}
	return tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, cfg)
}

func clientTLSConfig(tlsConfigPB *clientpb.TLS) (*tls.Config, error) {
	if tlsConfigPB == nil || !tlsConfigPB.GetEnable() {
		return nil, nil
	}

	rootCAs := x509.NewCertPool()
	if ca := tlsConfigPB.GetCa(); ca != nil && ca.GetCert() != "" {
		if ok := rootCAs.AppendCertsFromPEM([]byte(ca.GetCert())); !ok {
			return nil, errors.New("append tls ca cert failed")
		}
	}

	return &tls.Config{
		RootCAs:    rootCAs,
		MinVersion: tls.VersionTLS13,
	}, nil
}

func findSpiteByName(spites []*implantpb.Spite, name string) *implantpb.Spite {
	for _, spite := range spites {
		if spite == nil {
			continue
		}
		if spite.GetName() == name {
			return spite
		}
	}
	return nil
}

func trimHTTPBodyEnvelope(body, prefix, suffix []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, nil
	}
	if len(prefix) > 0 {
		if !bytes.HasPrefix(body, prefix) {
			return nil, fmt.Errorf("response body missing prefix %q", string(prefix))
		}
		body = body[len(prefix):]
	}
	if len(suffix) > 0 {
		if !bytes.HasSuffix(body, suffix) {
			return nil, fmt.Errorf("response body missing suffix %q", string(suffix))
		}
		body = body[:len(body)-len(suffix)]
	}
	return body, nil
}

func requestCryptos(encryptions []*clientpb.Encryption) ([]cryptostream.Cryptor, error) {
	cryptos, err := configs.NewCrypto(encryptions)
	if err != nil {
		return nil, err
	}
	if len(cryptos) == 0 {
		return nil, errors.New("pipeline has no encryption configured")
	}
	return cryptos, nil
}
