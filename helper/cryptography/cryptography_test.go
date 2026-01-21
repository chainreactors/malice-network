package cryptography

import (
	"bytes"
	"crypto/rand"
	"fmt"
	insecureRand "math/rand"
	"os"
	"sync"
	"testing"

	implantCrypto "github.com/chainreactors/malice-network/helper/cryptography/implant"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
)

var (
	sample1 = randomData()
	sample2 = randomData()

	serverAgeKeyPair      *AgeKeyPair
	implantPeerAgeKeyPair *AgeKeyPair
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256)+10)
	rand.Read(buf)
	return buf
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	var err error
	serverAgeKeyPair, err = RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	implantPeerAgeKeyPair, err = RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	implantCrypto.SetSecrets(
		implantPeerAgeKeyPair.Public,
		implantPeerAgeKeyPair.Private,
		MinisignServerSign([]byte(implantPeerAgeKeyPair.Public)),
		serverAgeKeyPair.Public,
		MinisignServerPublicKey(),
	)
}

func TestAgeEncryptDecrypt(t *testing.T) {
	encrypted, err := AgeEncrypt(serverAgeKeyPair.Public, sample1)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := AgeDecrypt(serverAgeKeyPair.Private, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, decrypted) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

func TestAgeEncrypt(t *testing.T) {
	data := "Hello, Age encryption test!"
	encrypted, err := AgeEncrypt("age1xcc0cmmz7zez2le3dkutrzfwf7tuuxwt4weq7wrzfdrary5f89tq3rsp2r", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(encrypted)
	if !bytes.Equal([]byte(data), encrypted) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

func TestAgeDecrypt(t *testing.T) {
	data := []byte{97, 103, 101, 45, 101, 110, 99, 114, 121, 112, 116, 105, 111, 110, 46, 111, 114, 103, 47, 118, 49, 10, 45, 62, 32, 88, 50, 53, 53, 49, 57, 32, 112, 115, 88, 48, 103, 104, 65, 84, 68, 120, 77, 111, 97, 84, 87, 77, 48, 47, 83, 117, 119, 50, 80, 107, 114, 52, 66, 43, 88, 105, 89, 75, 54, 112, 81, 122, 112, 43, 86, 104, 116, 103, 85, 10, 51, 82, 89, 50, 54, 116, 119, 70, 111, 108, 101, 70, 121, 66, 110, 57, 66, 101, 47, 121, 69, 79, 102, 99, 119, 76, 56, 107, 111, 115, 57, 55, 52, 115, 117, 110, 52, 56, 108, 48, 119, 69, 69, 10, 45, 62, 32, 66, 45, 103, 114, 101, 97, 115, 101, 32, 116, 61, 63, 42, 123, 75, 42, 32, 44, 47, 10, 66, 103, 111, 85, 119, 76, 83, 69, 120, 74, 120, 74, 87, 85, 109, 71, 118, 53, 73, 51, 120, 70, 121, 76, 43, 113, 52, 57, 97, 117, 50, 86, 74, 118, 108, 89, 47, 75, 110, 98, 66, 65, 49, 108, 72, 56, 48, 48, 52, 112, 98, 89, 47, 71, 69, 99, 89, 53, 52, 10, 45, 45, 45, 32, 56, 106, 115, 65, 101, 57, 97, 69, 108, 110, 116, 50, 109, 67, 99, 103, 122, 82, 48, 113, 53, 116, 55, 118, 57, 90, 86, 98, 112, 90, 85, 85, 83, 77, 71, 55, 89, 50, 79, 86, 104, 88, 81, 10, 226, 0, 72, 213, 103, 70, 169, 21, 148, 223, 128, 36, 70, 193, 95, 18, 97, 75, 179, 247, 222, 134, 200, 37, 24, 71, 167, 217, 5, 2, 143, 49, 50, 111, 245, 43, 73, 220, 140, 30, 133, 253, 34, 169, 28, 42, 179, 41, 170, 121, 110, 133, 51, 13, 184, 144, 192, 157, 152, 232, 20, 247, 130, 113, 201, 129, 233, 236, 222, 218, 132, 55, 199, 115, 246, 2, 208, 37, 248, 92, 110, 250, 188, 82, 162, 169, 104, 254, 34, 150, 212, 237, 208, 206, 202, 69, 32, 21, 74, 112, 195, 59, 0, 161, 192, 219, 139, 233, 197, 157, 177, 174, 7, 84, 168, 28, 125, 18, 148, 94, 225, 173, 98, 197, 239, 250, 240, 252, 1, 139, 146, 64, 22, 247, 199, 12, 237, 63, 195, 64, 157, 168, 82, 35, 64, 253, 114, 176, 11, 216, 112, 187, 212, 217, 28, 249, 67, 33, 131, 22, 87, 246, 79, 52, 91, 107, 143, 210, 77, 150, 104, 48, 7, 86, 165, 103, 13, 188, 228, 193, 194, 246, 184, 85, 121, 73, 54, 177, 66, 145, 103, 47, 96, 134, 133, 85, 187, 66, 123, 141, 198, 182, 49, 195, 73, 71, 29, 152, 166, 176, 69, 124, 177, 249, 0, 242, 169, 169, 151, 64, 188, 45, 45, 109, 252, 215, 94, 188, 112, 245, 5, 182, 50, 42, 203, 55, 133, 166, 160, 209, 159, 127, 167, 132, 222, 84, 108, 108, 19, 237, 154, 20, 109, 118, 175, 120, 75, 216, 206, 41, 246, 68, 110, 190, 132, 138, 151, 202, 203, 118, 232, 245, 158, 57, 159, 191, 188, 94, 173, 76, 214, 55, 75, 62, 94, 66, 185, 3, 42, 193, 217, 142, 136, 219, 175, 116, 107, 148, 157, 165, 210, 216, 71, 206, 237, 83, 106, 236, 52, 216, 124, 216, 13, 168, 53, 137, 180, 197, 156, 55, 156, 185, 70, 189, 47, 71, 160, 204, 158, 49, 16, 238, 127, 191, 31, 252, 229, 210, 227, 7, 151, 157, 146, 168, 115, 56, 223, 6, 253, 44, 170, 49, 236, 217, 55, 187, 248, 224, 222, 162, 181, 46, 225, 189, 197, 98, 251, 135, 185, 180, 138, 71, 218, 247, 96, 71, 91, 158, 186, 158, 86, 229, 226, 82, 3, 5, 237, 177, 176, 132, 17, 97, 227, 49, 217, 7, 195, 149, 130, 114, 36, 76, 64, 134, 254, 21, 116, 249, 103, 250, 111, 154, 249, 176, 209, 62, 65, 254, 216, 50, 113, 61, 53, 43, 36, 224, 244, 101, 181, 186, 198, 27, 74, 63, 146, 119, 108, 98, 236, 16, 156, 44, 60, 132, 173, 82, 31, 205, 167, 186, 249, 2, 123, 68, 86, 94, 80, 112, 165, 116, 76, 87, 25, 116, 2, 250, 212, 231, 254, 14, 130, 18, 175, 10, 198, 204, 178, 73, 68, 214, 6, 30, 16, 251, 243, 199, 47, 125, 212, 110, 36, 80, 5, 42, 253, 33, 27, 179, 50, 53, 130, 152, 75, 0, 79, 84, 160, 179, 238, 179, 203, 248, 183, 103, 83, 53, 18, 181, 80, 120, 171, 110, 142, 68, 58, 52, 220, 163, 44, 205, 124, 215, 86, 101, 6, 83, 177, 250, 183, 115, 213, 236, 226, 185, 143, 251, 73, 71, 117, 34, 57, 122, 236, 150, 230, 40, 219, 122, 237, 35, 116, 7, 88, 190, 205, 124, 42, 147, 135, 252, 194, 156, 188, 228, 102, 238, 162, 127, 12, 204, 8, 56, 119, 201, 158, 225, 15, 140, 149, 187, 207, 64, 210, 35, 96, 18, 165, 22, 54, 170, 199, 51, 49, 154, 215, 220, 3, 153, 109, 91, 145, 237, 136, 74, 12, 207, 195, 25, 152, 108, 175, 9, 185, 194, 50, 117, 31, 181, 79, 77, 45, 147, 39, 80, 49, 80, 153, 118, 42, 199, 74, 207, 111, 0, 107, 14, 12, 171, 240, 186, 52, 73, 25, 133, 5, 91, 165, 44, 207, 37, 142, 177, 104, 23, 71, 234, 80, 110, 254, 110, 199, 162, 204, 194, 193, 28, 149, 222, 47, 26, 204, 186, 192, 23, 204, 166, 194, 14, 58, 20, 102, 233, 123, 128, 205, 122, 206, 25, 96, 254, 101, 55, 83, 113, 117, 77, 207, 34, 166, 231, 253, 191, 218, 177, 24, 227, 92, 9, 166, 228, 217, 238, 7, 66, 65, 218, 202, 91, 225, 203, 183, 29, 87, 168, 76, 255, 186, 204, 199, 245, 85, 90, 149, 38, 208, 70, 31, 28, 202, 92, 7, 106, 158, 50, 186, 23, 179, 29, 85, 234, 104, 245, 21, 186, 167, 37, 50, 10, 184, 119, 246, 96, 62, 201, 43, 125, 128, 239, 79, 163, 5, 116, 45, 149, 27, 147, 181, 121, 243, 143, 31, 193, 21, 91, 5, 107, 179, 114, 159, 161, 66, 47, 52, 24, 103, 249, 242, 140, 12, 17, 96, 8, 116, 222, 56, 117, 126, 83, 184, 22, 186, 190, 175, 226, 160, 97, 18, 222, 193, 84, 245, 29, 195, 81, 228, 140, 223, 123, 218, 124, 245, 214, 6, 131, 253, 194, 134, 169, 45, 4, 158, 192, 175, 71, 205, 207, 31, 32, 141, 53, 117, 170, 218, 15, 72, 102, 211, 105}
	_, err := AgeDecrypt("AGE-SECRET-KEY-1G0VT6PZP0P3CHK9HR0W8J7EF04DWP9TWH07MR27CCFVXR8HDJJTQU2DFRN", data)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeTamperEncryptDecrypt(t *testing.T) {
	encrypted, err := AgeEncrypt(serverAgeKeyPair.Public, sample1)
	if err != nil {
		t.Fatal(err)
	}
	encrypted[insecureRand.Intn(len(encrypted))] ^= 0xFF
	_, err = AgeDecrypt(serverAgeKeyPair.Private, encrypted)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeWrongKeyEncryptDecrypt(t *testing.T) {
	encrypted, err := AgeEncrypt(serverAgeKeyPair.Public, sample1)
	if err != nil {
		t.Fatal(err)
	}
	keyPair, _ := RandomAgeKeyPair()
	_, err = AgeDecrypt(keyPair.Private, encrypted)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeKeyEx(t *testing.T) {
	sessionKey := RandomSymmetricKey()
	plaintext := sessionKey[:]
	ciphertext, err := implantCrypto.AgeKeyExToServer(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext[32:], // Remove prepended public key hash
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Session key does not match")
	}
}

func TestAgeKeyExTamper(t *testing.T) {
	sessionKey := RandomSymmetricKey()
	plaintext := sessionKey[:]
	allCiphertext, err := implantCrypto.AgeKeyExToServer(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the ciphertext
	ciphertext := allCiphertext[32:]
	ciphertext[insecureRand.Intn(len(ciphertext))] ^= 0xFF
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext,
	)
	if err == nil {
		t.Fatal(err)
	}

	// Leave an invalid header with valid ciphertext
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		allCiphertext,
	)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeKeyExReplay(t *testing.T) {
	sessionKey := RandomSymmetricKey()
	plaintext := sessionKey[:]
	allCiphertext, err := implantCrypto.AgeKeyExToServer(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	ciphertext := allCiphertext[32:]
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext,
	)
	if err == nil {
		t.Fatal(err)
	}
}

// TestEncryptDecrypt - Test AEAD functions
func TestEncryptDecrypt(t *testing.T) {
	key := RandomSymmetricKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := Decrypt(key, cipher1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Fatalf("Sample does not match decrypted data")
	}

	key = RandomSymmetricKey()
	cipher2, err := Encrypt(key, sample2)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := Decrypt(key, cipher2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

// TestTamperData - Detect tampered ciphertext
func TestTamperData(t *testing.T) {
	key := RandomSymmetricKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}

	index := insecureRand.Intn(len(cipher1))
	cipher1[index]++

	_, err = Decrypt(key, cipher1)
	if err == nil {
		t.Fatalf("Decrypted tampered data, should have resulted in Fatal")
	}
}

// TestWrongKey - Attempt to decrypt with wrong key
func TestWrongKey(t *testing.T) {
	key := RandomSymmetricKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	key2 := RandomSymmetricKey()
	_, err = Decrypt(key2, cipher1)
	if err == nil {
		t.Fatalf("Decrypted with wrong key, should have resulted in Fatal")
	}
}

// TestCipherContext - Test CipherContext
func TestCipherContext(t *testing.T) {
	testKey := RandomSymmetricKey()
	cipherCtx1 := &CipherContext{
		Key:    testKey,
		replay: &sync.Map{},
	}
	cipherCtx2 := implantCrypto.NewCipherContext(testKey)

	sample := randomData()

	ciphertext, err := cipherCtx1.Encrypt(sample)
	if err != nil {
		t.Fatalf("Failed to encrypt sample: %s", err)
	}
	_, err = cipherCtx1.Decrypt(ciphertext[minisign.RawSigSize:])
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack (1)")
	}
	_, err = cipherCtx1.Decrypt(ciphertext[minisign.RawSigSize:])
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack (2)")
	}

	plaintext, err := cipherCtx2.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample, plaintext) {
		t.Fatalf("Sample does not match decrypted data")
	}
	_, err = cipherCtx2.Decrypt(ciphertext)
	if err != implantCrypto.ErrReplayAttack {
		t.Fatal("Failed to detect replay attack (3)")
	}
}

// TestEncryptDecrypt - Test AEAD functions
func TestImplantEncryptDecrypt(t *testing.T) {
	key := RandomSymmetricKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := implantCrypto.Decrypt(key, cipher1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Fatalf("Sample does not match decrypted data")
	}

	key = RandomSymmetricKey()
	cipher2, err := implantCrypto.Encrypt(key, sample2)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := Decrypt(key, cipher2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

func TestServerMinisign(t *testing.T) {
	message := randomData()
	privateKey := MinisignServerPrivateKey()
	signature := minisign.Sign(*privateKey, message)
	if !minisign.Verify(privateKey.Public().(minisign.PublicKey), message, signature) {
		t.Fatalf("Failed to very message with server minisign")
	}
	message[0]++
	if minisign.Verify(privateKey.Public().(minisign.PublicKey), message, signature) {
		t.Fatalf("Minisign verified tampered message")
	}
}

func TestImplantMinisign(t *testing.T) {
	message := randomData()
	privateKey := MinisignServerPrivateKey()
	signature := minisign.Sign(*privateKey, message)

	publicKey := privateKey.Public().(minisign.PublicKey)
	publicKeyTxt, err := publicKey.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	implantPublicKey, err := implantCrypto.DecodeMinisignPublicKey(string(publicKeyTxt))
	if err != nil {
		t.Fatal(err)
	}
	implantSig, err := implantCrypto.DecodeMinisignSignature(string(signature))
	if err != nil {
		t.Fatal(err)
	}
	valid, err := implantPublicKey.Verify(message, implantSig)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("Implant failed to verify minisign signature")
	}
	message[0]++
	valid, err = implantPublicKey.Verify(message, implantSig)
	if err == nil {
		t.Fatal("Expected invalid signature error")
	}
	if valid {
		t.Fatal("Implant verified tampered message")
	}

}
