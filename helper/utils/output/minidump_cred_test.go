package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/malice-network/helper/sekurlsa"
)

func TestCredentialContextsFromSekurlsa(t *testing.T) {
	var nt [16]byte
	copy(nt[:], []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef})
	res := &sekurlsa.Result{
		Sessions: []sekurlsa.LogonSession{
			{
				UserName:    "alice",
				LogonDomain: "CORP",
				Credentials: []sekurlsa.Credential{
					&sekurlsa.MSVCredential{UserName: "alice", LogonDomain: "CORP", NTHash: nt, Found: true},
					&sekurlsa.WdigestCredential{UserName: "alice", LogonDomain: "CORP", Password: "P@ssw0rd", Found: true},
					&sekurlsa.MSVCredential{UserName: "empty", Found: true},
				},
			},
		},
	}
	creds := credentialContextsFromSekurlsa(res)
	if len(creds) != 2 {
		t.Fatalf("got %d creds, want 2", len(creds))
	}
	if creds[0].CredentialType != NtlmCredential || creds[0].Params["password"] != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("ntlm cred = %+v", creds[0])
	}
	if creds[0].Params["source"] != "nanodump" || creds[0].Target != `CORP\alice` {
		t.Fatalf("ntlm target/source = %+v", creds[0])
	}
	if creds[1].CredentialType != UserPassCredential || creds[1].Params["password"] != "P@ssw0rd" {
		t.Fatalf("password cred = %+v", creds[1])
	}
}

func TestParseMinidumpFileSkipsNonDump(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	creds, err := ParseMinidumpFile(path)
	if err != nil {
		t.Fatalf("ParseMinidumpFile: %v", err)
	}
	if creds != nil {
		t.Fatalf("creds = %v, want nil", creds)
	}
}
