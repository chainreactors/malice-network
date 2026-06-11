package output

import "testing"

func TestParseHashdumpBOFOutput(t *testing.T) {
	content := []byte(`[HASHDUMP] Dumped SAM and SYSTEM[HASHDUMP] Found current control set: 1
[HASHDUMP] Bootkey: 00000000000000000000000000000000[HASHDUMP] Decrypted bootkey: 11111111111111111111111111111111Administrator:500:<empty>
Guest:501:<empty>
DefaultAccount:503:<empty>
svc_backup:1001:0123456789ABCDEF0123456789ABCDEF
demo_user:1002:FEDCBA9876543210FEDCBA9876543210`)

	creds, err := ParseHashdump(content)
	if err != nil {
		t.Fatalf("ParseHashdump failed: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("ParseHashdump returned %d creds, want 2", len(creds))
	}

	tests := []struct {
		index    int
		user     string
		rid      string
		password string
	}{
		{index: 0, user: "svc_backup", rid: "1001", password: "0123456789abcdef0123456789abcdef"},
		{index: 1, user: "demo_user", rid: "1002", password: "fedcba9876543210fedcba9876543210"},
	}
	for _, tt := range tests {
		cred := creds[tt.index]
		if cred.CredentialType != NtlmCredential {
			t.Fatalf("cred[%d] type = %q, want %q", tt.index, cred.CredentialType, NtlmCredential)
		}
		if cred.Target != "SAM\\"+tt.user {
			t.Fatalf("cred[%d] target = %q, want %q", tt.index, cred.Target, "SAM\\"+tt.user)
		}
		if cred.Params["username"] != tt.user {
			t.Fatalf("cred[%d] username = %q, want %q", tt.index, cred.Params["username"], tt.user)
		}
		if cred.Params["rid"] != tt.rid {
			t.Fatalf("cred[%d] rid = %q, want %q", tt.index, cred.Params["rid"], tt.rid)
		}
		if cred.Params["password"] != tt.password {
			t.Fatalf("cred[%d] password = %q, want %q", tt.index, cred.Params["password"], tt.password)
		}
		if cred.Params["source"] != "hashdump" {
			t.Fatalf("cred[%d] source = %q, want hashdump", tt.index, cred.Params["source"])
		}
	}
}
