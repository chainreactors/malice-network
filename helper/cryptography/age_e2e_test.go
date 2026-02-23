package cryptography

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/proto"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

// TestAgeE2ERoundTrip verifies basic age encrypt/decrypt round-trip
func TestAgeE2ERoundTrip(t *testing.T) {
	keyPair, err := RandomAgeKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Verify key format
	if !strings.HasPrefix(keyPair.Public, "age1") {
		t.Fatalf("public key must start with 'age1', got: %s", keyPair.Public[:10])
	}
	if !strings.HasPrefix(keyPair.Private, "AGE-SECRET-KEY-1") {
		t.Fatalf("private key must start with 'AGE-SECRET-KEY-1', got: %s", keyPair.Private[:16])
	}

	plaintext := []byte("Hello, age E2E test!")
	encrypted, err := AgeEncrypt(keyPair.Public, plaintext)
	if err != nil {
		t.Fatalf("AgeEncrypt failed: %v", err)
	}

	// VERIFY: ciphertext must differ from plaintext
	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("BUG: ciphertext is identical to plaintext — encryption did not execute")
	}

	// VERIFY: ciphertext must be larger (age header + AEAD overhead)
	if len(encrypted) <= len(plaintext) {
		t.Fatalf("BUG: ciphertext (%d bytes) is not larger than plaintext (%d bytes)",
			len(encrypted), len(plaintext))
	}

	// VERIFY: ciphertext must contain age header
	if !bytes.Contains(encrypted, []byte("age-encryption.org/v1")) {
		t.Fatal("BUG: ciphertext does not contain age v1 header")
	}

	decrypted, err := AgeDecrypt(keyPair.Private, encrypted)
	if err != nil {
		t.Fatalf("AgeDecrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("round-trip failed: decrypted data does not match")
	}
}

// TestAgeE2EBidirectional verifies two-keypair bidirectional communication
func TestAgeE2EBidirectional(t *testing.T) {
	serverKP, err := RandomAgeKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	implantKP, err := RandomAgeKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the two keypairs are distinct
	if serverKP.Public == implantKP.Public {
		t.Fatal("BUG: server and implant generated identical public keys")
	}

	// Implant -> Server: encrypt with server_pub, decrypt with server_priv
	msgToServer := []byte("implant checkin data")
	enc, err := AgeEncrypt(serverKP.Public, msgToServer)
	if err != nil {
		t.Fatalf("encrypt to server: %v", err)
	}

	// VERIFY: wrong key fails
	_, err = AgeDecrypt(implantKP.Private, enc)
	if err == nil {
		t.Fatal("BUG: implant private key decrypted data encrypted for server")
	}

	// Correct key succeeds
	dec, err := AgeDecrypt(serverKP.Private, enc)
	if err != nil {
		t.Fatalf("server decrypt: %v", err)
	}
	if !bytes.Equal(dec, msgToServer) {
		t.Fatal("implant->server direction: data mismatch")
	}

	// Server -> Implant: encrypt with implant_pub, decrypt with implant_priv
	msgToImplant := []byte("server tasking payload")
	enc, err = AgeEncrypt(implantKP.Public, msgToImplant)
	if err != nil {
		t.Fatalf("encrypt to implant: %v", err)
	}

	// VERIFY: wrong key fails
	_, err = AgeDecrypt(serverKP.Private, enc)
	if err == nil {
		t.Fatal("BUG: server private key decrypted data encrypted for implant")
	}

	// Correct key succeeds
	dec, err = AgeDecrypt(implantKP.Private, enc)
	if err != nil {
		t.Fatalf("implant decrypt: %v", err)
	}
	if !bytes.Equal(dec, msgToImplant) {
		t.Fatal("server->implant direction: data mismatch")
	}
}

// TestAgeE2EKeyExchangeProtocol simulates the full key exchange flow
func TestAgeE2EKeyExchangeProtocol(t *testing.T) {
	// Phase 1: Initial key distribution (from pipeline)
	serverKP, _ := RandomAgeKeyPair()
	implantKP, _ := RandomAgeKeyPair()

	// Verify initial bidirectional communication
	verifyBidirectionalStrict(t, serverKP, implantKP, "initial")

	// Phase 2: Server triggers key exchange - generates new server keypair
	newServerKP, _ := RandomAgeKeyPair()

	// Server sends KeyExchangeRequest with new_server_pub
	// (encrypted to implant using current implant_pub)
	reqPayload := []byte(newServerKP.Public)
	encReq, err := AgeEncrypt(implantKP.Public, reqPayload)
	if err != nil {
		t.Fatalf("encrypt key exchange request: %v", err)
	}

	// VERIFY: encrypted payload does not contain raw public key
	if bytes.Contains(encReq, []byte(newServerKP.Public)) {
		t.Fatal("BUG: encrypted key exchange request contains raw public key in cleartext")
	}

	decReq, err := AgeDecrypt(implantKP.Private, encReq)
	if err != nil {
		t.Fatalf("implant decrypt key exchange request: %v", err)
	}
	receivedNewServerPub := string(decReq)
	if receivedNewServerPub != newServerKP.Public {
		t.Fatal("implant did not receive correct new server public key")
	}

	// Phase 3: Implant generates new keypair and responds
	newImplantKP, _ := RandomAgeKeyPair()

	// Implant sends KeyExchangeResponse with new_implant_pub
	respPayload := []byte(newImplantKP.Public)
	encResp, err := AgeEncrypt(serverKP.Public, respPayload)
	if err != nil {
		t.Fatalf("encrypt key exchange response: %v", err)
	}
	decResp, err := AgeDecrypt(serverKP.Private, encResp)
	if err != nil {
		t.Fatalf("server decrypt key exchange response: %v", err)
	}
	receivedNewImplantPub := string(decResp)
	if receivedNewImplantPub != newImplantKP.Public {
		t.Fatal("server did not receive correct new implant public key")
	}

	// Phase 4+5: Verify new keys work, old keys fail
	verifyBidirectionalStrict(t, newServerKP, newImplantKP, "post-rotation")

	// Phase 6: old keys MUST NOT decrypt new-key data
	encNew, _ := AgeEncrypt(newServerKP.Public, []byte("new data"))
	_, err = AgeDecrypt(serverKP.Private, encNew)
	if err == nil {
		t.Fatal("old server key must not decrypt new-key data")
	}
	encNew2, _ := AgeEncrypt(newImplantKP.Public, []byte("new data 2"))
	_, err = AgeDecrypt(implantKP.Private, encNew2)
	if err == nil {
		t.Fatal("old implant key must not decrypt new-key data")
	}
}

// TestAgeE2EKeyUpdateAtomicity simulates Bug 4 scenario
func TestAgeE2EKeyUpdateAtomicity(t *testing.T) {
	// Initial keys
	serverKP, _ := RandomAgeKeyPair()
	implantKP, _ := RandomAgeKeyPair()

	// VERIFY: initial communication works
	verifyBidirectionalStrict(t, serverKP, implantKP, "pre-exchange")

	// Key exchange: new keypairs generated
	newServerKP, _ := RandomAgeKeyPair()
	newImplantKP, _ := RandomAgeKeyPair()

	// Simulate Bug 4: server updates private key first, public key later
	// After UpdatePrivateKey but before UpdatePublicKey:
	// server has: private = new_server_priv, public = old_implant_pub (STALE!)
	_ = serverKP // old server keypair no longer used after rotation
	brokenServerKP := &AgeKeyPair{
		Public:  implantKP.Public,    // still old
		Private: newServerKP.Private, // already new
	}

	// Meanwhile implant has already updated both keys
	updatedImplantKP := newImplantKP

	// Server tries to send to implant using stale public key (old implant pub)
	msg := []byte("message during race window")
	encRace, err := AgeEncrypt(brokenServerKP.Public, msg) // uses OLD implant pub
	if err != nil {
		t.Fatalf("encrypt during race: %v", err)
	}

	// VERIFY: the old implant key CAN decrypt (since encrypted for old key)
	decOldKey, err := AgeDecrypt(implantKP.Private, encRace)
	if err != nil {
		t.Fatalf("old implant key should decrypt data encrypted for it: %v", err)
	}
	if !bytes.Equal(decOldKey, msg) {
		t.Fatal("old implant key decrypted wrong data")
	}

	// VERIFY: new implant key CANNOT decrypt (encrypted for old key, not new)
	_, err = AgeDecrypt(updatedImplantKP.Private, encRace)
	if err == nil {
		t.Fatal("BUG: new implant key decrypted data encrypted for OLD key — " +
			"this means the race condition bug would NOT be caught")
	}

	// Now simulate correct fix: both keys updated atomically
	fixedServerKP := &AgeKeyPair{
		Public:  newImplantKP.Public, // both updated together
		Private: newServerKP.Private,
	}

	// Server sends with correct new implant public key
	encFixed, err := AgeEncrypt(fixedServerKP.Public, msg)
	if err != nil {
		t.Fatalf("encrypt with fixed keys: %v", err)
	}

	// VERIFY: new implant key CAN decrypt
	dec, err := AgeDecrypt(updatedImplantKP.Private, encFixed)
	if err != nil {
		t.Fatalf("decrypt with fixed keys: %v", err)
	}
	if !bytes.Equal(dec, msg) {
		t.Fatal("fixed atomic update: data mismatch")
	}
}

// TestAgeE2EProtobufIntegration tests protobuf + snappy + age pipeline
// Simulates the Rust implant's marshal flow and Go server's parse flow
func TestAgeE2EProtobufIntegration(t *testing.T) {
	serverKP, _ := RandomAgeKeyPair()
	implantKP, _ := RandomAgeKeyPair()

	// Build a Spites message
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{
				Name:   "test_module",
				TaskId: 42,
				Async:  true,
				Status: &implantpb.Status{TaskId: 42, Status: 0},
			},
		},
	}

	// === Implant -> Server ===
	// Step 1: protobuf marshal
	pbBytes, err := proto.Marshal(spites)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	t.Logf("protobuf size: %d bytes", len(pbBytes))

	// Step 2: snappy compress
	compressed := snappy.Encode(nil, pbBytes)
	t.Logf("snappy compressed size: %d bytes", len(compressed))

	// Step 3: age encrypt with server's public key
	encrypted, err := AgeEncrypt(serverKP.Public, compressed)
	if err != nil {
		t.Fatalf("AgeEncrypt: %v", err)
	}
	t.Logf("age encrypted size: %d bytes", len(encrypted))

	// VERIFY: encrypted data is larger than compressed data
	if len(encrypted) <= len(compressed) {
		t.Fatalf("BUG: encrypted (%d) not larger than compressed (%d) — encryption may not have run",
			len(encrypted), len(compressed))
	}

	// VERIFY: encrypted data cannot be snappy-decoded directly
	_, decodeErr := snappy.Decode(nil, encrypted)
	if decodeErr == nil {
		t.Fatal("BUG: encrypted data is directly snappy-decodable — encryption may not have run")
	}

	// VERIFY: wrong key fails
	_, err = AgeDecrypt(implantKP.Private, encrypted)
	if err == nil {
		t.Fatal("BUG: implant key decrypted server-bound data")
	}

	// Step 4: age decrypt with server's private key
	decrypted, err := AgeDecrypt(serverKP.Private, encrypted)
	if err != nil {
		t.Fatalf("AgeDecrypt: %v", err)
	}

	// VERIFY: decrypted matches original compressed
	if !bytes.Equal(decrypted, compressed) {
		t.Fatal("decrypted data does not match original compressed data")
	}

	// Step 5: snappy decompress
	decompressed, err := snappy.Decode(nil, decrypted)
	if err != nil {
		t.Fatalf("snappy.Decode: %v", err)
	}

	// VERIFY: decompressed matches original protobuf bytes
	if !bytes.Equal(decompressed, pbBytes) {
		t.Fatal("decompressed data does not match original protobuf bytes")
	}

	// Step 6: protobuf unmarshal
	recovered := &implantpb.Spites{}
	if err := proto.Unmarshal(decompressed, recovered); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if len(recovered.Spites) != 1 {
		t.Fatalf("expected 1 spite, got %d", len(recovered.Spites))
	}
	if recovered.Spites[0].TaskId != 42 {
		t.Fatalf("expected task_id=42, got %d", recovered.Spites[0].TaskId)
	}
	if recovered.Spites[0].Name != "test_module" {
		t.Fatalf("expected name=test_module, got %s", recovered.Spites[0].Name)
	}

	// === Server -> Implant (reverse direction) ===
	response := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{Name: "server_task", TaskId: 99, Async: true},
		},
	}

	respBytes, _ := proto.Marshal(response)
	respCompressed := snappy.Encode(nil, respBytes)
	respEncrypted, err := AgeEncrypt(implantKP.Public, respCompressed)
	if err != nil {
		t.Fatalf("server AgeEncrypt: %v", err)
	}

	// VERIFY: wrong key fails for reverse direction too
	_, err = AgeDecrypt(serverKP.Private, respEncrypted)
	if err == nil {
		t.Fatal("BUG: server key decrypted implant-bound data")
	}

	respDecrypted, err := AgeDecrypt(implantKP.Private, respEncrypted)
	if err != nil {
		t.Fatalf("implant AgeDecrypt: %v", err)
	}
	if !bytes.Equal(respDecrypted, respCompressed) {
		t.Fatal("reverse direction: decrypted does not match compressed")
	}

	respDecompressed, err := snappy.Decode(nil, respDecrypted)
	if err != nil {
		t.Fatalf("implant snappy.Decode: %v", err)
	}
	respRecovered := &implantpb.Spites{}
	if err := proto.Unmarshal(respDecompressed, respRecovered); err != nil {
		t.Fatalf("implant proto.Unmarshal: %v", err)
	}
	if respRecovered.Spites[0].TaskId != 99 {
		t.Fatalf("expected task_id=99, got %d", respRecovered.Spites[0].TaskId)
	}
}

// TestAgeE2EProtobufKeyRotation combines protobuf+snappy+age with key rotation
func TestAgeE2EProtobufKeyRotation(t *testing.T) {
	serverKP, _ := RandomAgeKeyPair()
	implantKP, _ := RandomAgeKeyPair()

	makeMsg := func(taskID uint32, name string) *implantpb.Spites {
		return &implantpb.Spites{
			Spites: []*implantpb.Spite{
				{Name: name, TaskId: taskID, Async: true},
			},
		}
	}

	// sendAndVerify does the full pipeline and verifies encryption actually ran
	sendAndVerify := func(msg *implantpb.Spites, encPub, decPriv string, wrongPriv string) *implantpb.Spites {
		t.Helper()
		pbBytes, _ := proto.Marshal(msg)
		compressed := snappy.Encode(nil, pbBytes)
		encrypted, err := AgeEncrypt(encPub, compressed)
		if err != nil {
			t.Fatalf("AgeEncrypt: %v", err)
		}

		// VERIFY: encryption happened
		if bytes.Equal(encrypted, compressed) {
			t.Fatal("BUG: encrypted == compressed, encryption did not run")
		}

		// VERIFY: wrong key fails
		if wrongPriv != "" {
			_, err = AgeDecrypt(wrongPriv, encrypted)
			if err == nil {
				t.Fatal("BUG: wrong key decrypted the data")
			}
		}

		decrypted, err := AgeDecrypt(decPriv, encrypted)
		if err != nil {
			t.Fatalf("AgeDecrypt: %v", err)
		}
		decompressed, err := snappy.Decode(nil, decrypted)
		if err != nil {
			t.Fatalf("snappy.Decode: %v", err)
		}
		recovered := &implantpb.Spites{}
		if err := proto.Unmarshal(decompressed, recovered); err != nil {
			t.Fatalf("proto.Unmarshal: %v", err)
		}
		return recovered
	}

	// Pre-rotation: both directions work, cross-keys fail
	r := sendAndVerify(makeMsg(1, "pre_rot_i2s"), serverKP.Public, serverKP.Private, implantKP.Private)
	if r.Spites[0].TaskId != 1 || r.Spites[0].Name != "pre_rot_i2s" {
		t.Fatal("pre-rotation implant->server: content mismatch")
	}
	r = sendAndVerify(makeMsg(2, "pre_rot_s2i"), implantKP.Public, implantKP.Private, serverKP.Private)
	if r.Spites[0].TaskId != 2 || r.Spites[0].Name != "pre_rot_s2i" {
		t.Fatal("pre-rotation server->implant: content mismatch")
	}

	// Key rotation
	newServerKP, _ := RandomAgeKeyPair()
	newImplantKP, _ := RandomAgeKeyPair()

	// Post-rotation: new keys work, old keys fail
	r = sendAndVerify(makeMsg(3, "post_rot_i2s"), newServerKP.Public, newServerKP.Private, serverKP.Private)
	if r.Spites[0].TaskId != 3 || r.Spites[0].Name != "post_rot_i2s" {
		t.Fatal("post-rotation implant->server: content mismatch")
	}
	r = sendAndVerify(makeMsg(4, "post_rot_s2i"), newImplantKP.Public, newImplantKP.Private, implantKP.Private)
	if r.Spites[0].TaskId != 4 || r.Spites[0].Name != "post_rot_s2i" {
		t.Fatal("post-rotation server->implant: content mismatch")
	}

	// Explicit cross-key failure: new encrypted data, old decrypt key
	pbBytes, _ := proto.Marshal(makeMsg(5, "cross"))
	compressed := snappy.Encode(nil, pbBytes)
	encrypted, _ := AgeEncrypt(newServerKP.Public, compressed)
	_, err := AgeDecrypt(serverKP.Private, encrypted)
	if err == nil {
		t.Fatal("old server key must not decrypt new-key data")
	}
	encrypted2, _ := AgeEncrypt(newImplantKP.Public, compressed)
	_, err = AgeDecrypt(implantKP.Private, encrypted2)
	if err == nil {
		t.Fatal("old implant key must not decrypt new-key data")
	}
}

// === TLV Wire Format Tests ===
// These tests verify the full TLV wire format (0xd1 + session_id + length + data + 0xd2)
// matching the Rust SpiteData pack/unpack format.

const (
	tlvStartDelimiter = 0xd1
	tlvEndDelimiter   = 0xd2
	tlvHeaderLength   = 9 // 1 (start) + 4 (session_id) + 4 (length)
)

// tlvPack constructs a TLV wire packet matching Rust SpiteData::pack()
func tlvPack(sessionID uint32, data []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(tlvStartDelimiter)
	_ = binary.Write(&buf, binary.LittleEndian, sessionID)
	_ = binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	buf.Write(data)
	buf.WriteByte(tlvEndDelimiter)
	return buf.Bytes()
}

// tlvUnpack parses a TLV wire packet matching Rust SpiteData::unpack()
// Returns session_id, data, error
func tlvUnpack(wire []byte) (uint32, []byte, error) {
	if len(wire) < tlvHeaderLength+1 {
		return 0, nil, fmt.Errorf("packet too short: %d bytes", len(wire))
	}
	if wire[0] != tlvStartDelimiter {
		return 0, nil, fmt.Errorf("invalid start delimiter: 0x%02x", wire[0])
	}
	if wire[len(wire)-1] != tlvEndDelimiter {
		return 0, nil, fmt.Errorf("invalid end delimiter: 0x%02x", wire[len(wire)-1])
	}
	sessionID := binary.LittleEndian.Uint32(wire[1:5])
	length := binary.LittleEndian.Uint32(wire[5:9])
	if int(length) != len(wire)-tlvHeaderLength-1 {
		return 0, nil, fmt.Errorf("length mismatch: header says %d, actual %d",
			length, len(wire)-tlvHeaderLength-1)
	}
	data := wire[tlvHeaderLength : tlvHeaderLength+int(length)]
	return sessionID, data, nil
}

// TestTLVWireFormatStructure verifies the TLV wire format byte layout
// matches the Rust SpiteData format: 0xd1 + session_id(4,LE) + length(4,LE) + data + 0xd2
func TestTLVWireFormatStructure(t *testing.T) {
	sessionID := uint32(0xDEADBEEF)
	payload := []byte("test payload data")

	wire := tlvPack(sessionID, payload)

	// VERIFY: start delimiter
	if wire[0] != 0xd1 {
		t.Fatalf("expected start delimiter 0xd1, got 0x%02x", wire[0])
	}

	// VERIFY: session_id at bytes 1-4 (Little-Endian)
	sid := binary.LittleEndian.Uint32(wire[1:5])
	if sid != sessionID {
		t.Fatalf("expected session_id 0x%08x, got 0x%08x", sessionID, sid)
	}

	// VERIFY: length at bytes 5-8 (Little-Endian)
	length := binary.LittleEndian.Uint32(wire[5:9])
	if length != uint32(len(payload)) {
		t.Fatalf("expected length %d, got %d", len(payload), length)
	}

	// VERIFY: data at bytes 9 to 9+length
	if !bytes.Equal(wire[9:9+length], payload) {
		t.Fatal("payload data mismatch in wire bytes")
	}

	// VERIFY: end delimiter
	if wire[len(wire)-1] != 0xd2 {
		t.Fatalf("expected end delimiter 0xd2, got 0x%02x", wire[len(wire)-1])
	}

	// VERIFY: total length = header(9) + data + end(1)
	expectedLen := tlvHeaderLength + len(payload) + 1
	if len(wire) != expectedLen {
		t.Fatalf("expected wire length %d, got %d", expectedLen, len(wire))
	}

	// VERIFY: round-trip unpack
	unpSID, unpData, err := tlvUnpack(wire)
	if err != nil {
		t.Fatalf("unpack failed: %v", err)
	}
	if unpSID != sessionID {
		t.Fatalf("unpack session_id mismatch: 0x%08x vs 0x%08x", unpSID, sessionID)
	}
	if !bytes.Equal(unpData, payload) {
		t.Fatal("unpack data mismatch")
	}
}

// TestTLVProtobufAgeRoundTrip tests the complete pipeline:
// protobuf → snappy → age encrypt → TLV pack → TLV unpack → age decrypt → snappy → protobuf
// This matches the Rust SpiteData marshal/parse + pack/unpack flow.
func TestTLVProtobufAgeRoundTrip(t *testing.T) {
	serverKP, _ := RandomAgeKeyPair()
	implantKP, _ := RandomAgeKeyPair()
	sessionID := uint32(0xAABBCCDD)

	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{Name: "test_module", TaskId: 42, Async: true,
				Status: &implantpb.Status{TaskId: 42, Status: 0}},
		},
	}

	// === Implant → Server (matches Rust marshal + pack) ===
	// Step 1: protobuf marshal
	pbBytes, err := proto.Marshal(spites)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	// Step 2: snappy compress
	compressed := snappy.Encode(nil, pbBytes)

	// Step 3: age encrypt with server's public key
	encrypted, err := AgeEncrypt(serverKP.Public, compressed)
	if err != nil {
		t.Fatalf("AgeEncrypt: %v", err)
	}

	// VERIFY: encryption actually occurred
	if bytes.Equal(encrypted, compressed) {
		t.Fatal("BUG: encrypted == compressed")
	}

	// Step 4: TLV pack
	wire := tlvPack(sessionID, encrypted)
	t.Logf("wire packet: %d bytes (header=%d, payload=%d, delimiters=2)",
		len(wire), tlvHeaderLength, len(encrypted))

	// VERIFY: wire starts with 0xd1, ends with 0xd2
	if wire[0] != 0xd1 || wire[len(wire)-1] != 0xd2 {
		t.Fatalf("wire delimiters: start=0x%02x end=0x%02x", wire[0], wire[len(wire)-1])
	}

	// === Server receives and parses (matches Go MaleficParser.ReadHeader + Parse) ===
	// Step 5: TLV unpack
	unpSID, unpData, err := tlvUnpack(wire)
	if err != nil {
		t.Fatalf("TLV unpack: %v", err)
	}
	if unpSID != sessionID {
		t.Fatalf("session_id mismatch: 0x%08x vs 0x%08x", unpSID, sessionID)
	}

	// VERIFY: unpacked data matches original encrypted data
	if !bytes.Equal(unpData, encrypted) {
		t.Fatal("unpacked data != original encrypted data")
	}

	// Step 6: age decrypt with server's private key
	decrypted, err := AgeDecrypt(serverKP.Private, unpData)
	if err != nil {
		t.Fatalf("AgeDecrypt: %v", err)
	}

	// VERIFY: wrong key fails
	_, err = AgeDecrypt(implantKP.Private, unpData)
	if err == nil {
		t.Fatal("BUG: implant key decrypted server-bound TLV data")
	}

	// Step 7: snappy decompress
	decompressed, err := snappy.Decode(nil, decrypted)
	if err != nil {
		t.Fatalf("snappy.Decode: %v", err)
	}

	// Step 8: protobuf unmarshal
	recovered := &implantpb.Spites{}
	if err := proto.Unmarshal(decompressed, recovered); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if len(recovered.Spites) != 1 {
		t.Fatalf("expected 1 spite, got %d", len(recovered.Spites))
	}
	if recovered.Spites[0].TaskId != 42 {
		t.Fatalf("expected task_id=42, got %d", recovered.Spites[0].TaskId)
	}
	if recovered.Spites[0].Name != "test_module" {
		t.Fatalf("expected name=test_module, got %s", recovered.Spites[0].Name)
	}

	// === Server → Implant (reverse direction, same TLV format) ===
	response := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{Name: "server_task", TaskId: 99, Async: true},
		},
	}
	respSID := uint32(0x11223344)
	respBytes, _ := proto.Marshal(response)
	respCompressed := snappy.Encode(nil, respBytes)
	respEncrypted, err := AgeEncrypt(implantKP.Public, respCompressed)
	if err != nil {
		t.Fatalf("server AgeEncrypt: %v", err)
	}
	respWire := tlvPack(respSID, respEncrypted)

	// Parse reverse direction
	unpRespSID, unpRespData, err := tlvUnpack(respWire)
	if err != nil {
		t.Fatalf("reverse TLV unpack: %v", err)
	}
	if unpRespSID != respSID {
		t.Fatalf("reverse session_id mismatch")
	}

	// VERIFY: wrong key fails
	_, err = AgeDecrypt(serverKP.Private, unpRespData)
	if err == nil {
		t.Fatal("BUG: server key decrypted implant-bound TLV data")
	}

	respDecrypted, err := AgeDecrypt(implantKP.Private, unpRespData)
	if err != nil {
		t.Fatalf("implant AgeDecrypt: %v", err)
	}
	respDecompressed, err := snappy.Decode(nil, respDecrypted)
	if err != nil {
		t.Fatalf("implant snappy.Decode: %v", err)
	}
	respRecovered := &implantpb.Spites{}
	if err := proto.Unmarshal(respDecompressed, respRecovered); err != nil {
		t.Fatalf("implant proto.Unmarshal: %v", err)
	}
	if respRecovered.Spites[0].TaskId != 99 {
		t.Fatalf("expected task_id=99, got %d", respRecovered.Spites[0].TaskId)
	}
}

// TestTLVKeyRotationFullPipeline tests key rotation with the complete
// TLV + protobuf + snappy + age pipeline
func TestTLVKeyRotationFullPipeline(t *testing.T) {
	serverKP, _ := RandomAgeKeyPair()
	implantKP, _ := RandomAgeKeyPair()

	marshalAndPack := func(spites *implantpb.Spites, sid uint32, encPub string) []byte {
		t.Helper()
		pbBytes, _ := proto.Marshal(spites)
		compressed := snappy.Encode(nil, pbBytes)
		encrypted, err := AgeEncrypt(encPub, compressed)
		if err != nil {
			t.Fatalf("AgeEncrypt: %v", err)
		}
		// VERIFY encryption occurred
		if bytes.Equal(encrypted, compressed) {
			t.Fatal("BUG: encrypted == compressed")
		}
		return tlvPack(sid, encrypted)
	}

	unpackAndParse := func(wire []byte, decPriv string) (uint32, *implantpb.Spites) {
		t.Helper()
		sid, data, err := tlvUnpack(wire)
		if err != nil {
			t.Fatalf("TLV unpack: %v", err)
		}
		decrypted, err := AgeDecrypt(decPriv, data)
		if err != nil {
			t.Fatalf("AgeDecrypt: %v", err)
		}
		decompressed, err := snappy.Decode(nil, decrypted)
		if err != nil {
			t.Fatalf("snappy.Decode: %v", err)
		}
		recovered := &implantpb.Spites{}
		if err := proto.Unmarshal(decompressed, recovered); err != nil {
			t.Fatalf("proto.Unmarshal: %v", err)
		}
		return sid, recovered
	}

	makeMsg := func(taskID uint32, name string) *implantpb.Spites {
		return &implantpb.Spites{
			Spites: []*implantpb.Spite{
				{Name: name, TaskId: taskID, Async: true},
			},
		}
	}

	// Pre-rotation: bidirectional communication via TLV
	wire1 := marshalAndPack(makeMsg(1, "pre_i2s"), 0x1111, serverKP.Public)
	sid1, r1 := unpackAndParse(wire1, serverKP.Private)
	if sid1 != 0x1111 || r1.Spites[0].TaskId != 1 || r1.Spites[0].Name != "pre_i2s" {
		t.Fatal("pre-rotation implant->server TLV: mismatch")
	}

	wire2 := marshalAndPack(makeMsg(2, "pre_s2i"), 0x2222, implantKP.Public)
	sid2, r2 := unpackAndParse(wire2, implantKP.Private)
	if sid2 != 0x2222 || r2.Spites[0].TaskId != 2 || r2.Spites[0].Name != "pre_s2i" {
		t.Fatal("pre-rotation server->implant TLV: mismatch")
	}

	// VERIFY: wrong key fails on TLV data
	_, data1, _ := tlvUnpack(wire1)
	_, err := AgeDecrypt(implantKP.Private, data1)
	if err == nil {
		t.Fatal("BUG: implant key decrypted server-bound TLV packet")
	}

	// Key rotation
	newServerKP, _ := RandomAgeKeyPair()
	newImplantKP, _ := RandomAgeKeyPair()

	// Post-rotation: new keys with TLV
	wire3 := marshalAndPack(makeMsg(3, "post_i2s"), 0x3333, newServerKP.Public)
	sid3, r3 := unpackAndParse(wire3, newServerKP.Private)
	if sid3 != 0x3333 || r3.Spites[0].TaskId != 3 || r3.Spites[0].Name != "post_i2s" {
		t.Fatal("post-rotation implant->server TLV: mismatch")
	}

	wire4 := marshalAndPack(makeMsg(4, "post_s2i"), 0x4444, newImplantKP.Public)
	sid4, r4 := unpackAndParse(wire4, newImplantKP.Private)
	if sid4 != 0x4444 || r4.Spites[0].TaskId != 4 || r4.Spites[0].Name != "post_s2i" {
		t.Fatal("post-rotation server->implant TLV: mismatch")
	}

	// VERIFY: old keys fail on post-rotation TLV data
	_, data3, _ := tlvUnpack(wire3)
	_, err = AgeDecrypt(serverKP.Private, data3)
	if err == nil {
		t.Fatal("old server key must not decrypt post-rotation TLV data")
	}
	_, data4, _ := tlvUnpack(wire4)
	_, err = AgeDecrypt(implantKP.Private, data4)
	if err == nil {
		t.Fatal("old implant key must not decrypt post-rotation TLV data")
	}
}

// TestTLVInvalidPackets verifies TLV parser rejects malformed packets
func TestTLVInvalidPackets(t *testing.T) {
	// Missing start delimiter
	_, _, err := tlvUnpack([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0xd2})
	if err == nil {
		t.Fatal("should reject packet with wrong start delimiter")
	}

	// Missing end delimiter
	_, _, err = tlvUnpack([]byte{0xd1, 0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err == nil {
		t.Fatal("should reject packet with wrong end delimiter")
	}

	// Packet too short
	_, _, err = tlvUnpack([]byte{0xd1, 0x01, 0x02})
	if err == nil {
		t.Fatal("should reject packet that is too short")
	}

	// Length mismatch (header says 10 bytes but only 0 bytes of data)
	_, _, err = tlvUnpack([]byte{0xd1, 0x01, 0x02, 0x03, 0x04, 0x0a, 0x00, 0x00, 0x00, 0xd2})
	if err == nil {
		t.Fatal("should reject packet with length mismatch")
	}

	// Valid empty payload
	sid, data, err := tlvUnpack(tlvPack(0x12345678, []byte{}))
	if err != nil {
		t.Fatalf("valid empty payload should parse: %v", err)
	}
	if sid != 0x12345678 {
		t.Fatalf("wrong session_id for empty payload")
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %d bytes", len(data))
	}
}

// verifyBidirectionalStrict verifies bidirectional communication with
// explicit cross-key failure checks to ensure encryption actually occurred
func verifyBidirectionalStrict(t *testing.T, serverKP, implantKP *AgeKeyPair, label string) {
	t.Helper()

	// Implant -> Server
	msg1 := []byte(label + " implant->server")
	enc, err := AgeEncrypt(serverKP.Public, msg1)
	if err != nil {
		t.Fatalf("[%s] encrypt to server: %v", label, err)
	}
	// VERIFY: wrong key fails
	_, err = AgeDecrypt(implantKP.Private, enc)
	if err == nil {
		t.Fatalf("[%s] BUG: implant key decrypted server-bound data", label)
	}
	// Correct key works
	dec, err := AgeDecrypt(serverKP.Private, enc)
	if err != nil {
		t.Fatalf("[%s] server decrypt: %v", label, err)
	}
	if !bytes.Equal(dec, msg1) {
		t.Fatalf("[%s] implant->server mismatch", label)
	}

	// Server -> Implant
	msg2 := []byte(label + " server->implant")
	enc, err = AgeEncrypt(implantKP.Public, msg2)
	if err != nil {
		t.Fatalf("[%s] encrypt to implant: %v", label, err)
	}
	// VERIFY: wrong key fails
	_, err = AgeDecrypt(serverKP.Private, enc)
	if err == nil {
		t.Fatalf("[%s] BUG: server key decrypted implant-bound data", label)
	}
	// Correct key works
	dec, err = AgeDecrypt(implantKP.Private, enc)
	if err != nil {
		t.Fatalf("[%s] implant decrypt: %v", label, err)
	}
	if !bytes.Equal(dec, msg2) {
		t.Fatalf("[%s] server->implant mismatch", label)
	}
}
