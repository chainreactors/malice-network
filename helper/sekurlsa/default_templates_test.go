package sekurlsa

import "testing"

// TestDefaultTemplates_RegisteredAtInit asserts that the canonical
// Win10/Win11 templates are reachable via templateFor() without any
// operator setup. This is the "no RegisterTemplate boilerplate"
// guarantee — a v0.23.x dump from one of the documented builds parses
// out of the box.
//
// Test calls registerDefaultTemplates() explicitly so it survives
// being run after a sibling test that called resetTemplates().
func TestDefaultTemplates_RegisteredAtInit(t *testing.T) {
	t.Cleanup(resetTemplates)
	resetTemplates()
	registerDefaultTemplates()

	cases := []struct {
		name  string
		build uint32
		want  bool
	}{
		// v0.25.2 reproduces KvcForensic's 9 build ranges plus pypykatz
		// LSA crypto offsets for Win 7 SP1 + Win 8.
		{"Win7 RTM", 7600, true},
		{"Win7 SP1 / Server 2008 R2", 7601, true},
		{"Win8 / Server 2012", 9200, true},
		{"Win8.1 / Server 2012 R2", 9600, true},
		{"Win10 RTM (1507)", 10240, true},
		{"Win10 1607 / Server 2016", 14393, true},
		{"Win10 1703", 15063, true},
		{"Win10 1709", 16299, true},
		{"Win10 1803", 17134, true},
		{"Win10 1809 / Server 2019", 17763, true},
		{"Win10 19H1 (1903)", 18362, true},
		{"Win10 1909", 18363, true},
		{"Win10 2004", 19041, true},
		{"Win10 21H2", 19044, true},
		{"Win10 22H2", 19045, true},
		{"Server 2022", 20348, true},
		{"Win11 21H2", 22000, true},
		{"Win11 22H2 pre-22622", 22621, true},
		{"Win11 22622", 22622, true},
		{"Win11 23H2", 22631, true},
		{"Win11 24H2 / Server 2025", 26100, true},
		{"Win11 25H2 (future)", 26200, true},
		{"far future", 999999, true},

		// Out-of-range: pre-NT6.1 (Vista, XP, Server 2003).
		{"Win XP / Server 2003 (NT 5.x)", 2600, false},
		{"Win Vista / Server 2008 (NT 6.0)", 6002, false},
		{"NT4 (just under 7600)", 7599, false},
	}

	for _, tc := range cases {
		got := templateFor(tc.build) != nil
		if got != tc.want {
			t.Errorf("templateFor(%d %s) coverage = %v, want %v",
				tc.build, tc.name, got, tc.want)
		}
	}
}

// TestDefaultTemplates_PassValidation confirms every shipping
// Template satisfies the same validate() guard that RegisterTemplate
// runs at registration. Catches drift if a future edit forgets a
// required pattern field.
func TestDefaultTemplates_PassValidation(t *testing.T) {
	for i, tpl := range builtinTemplates {
		if err := tpl.validate(); err != nil {
			t.Errorf("builtinTemplates[%d] (build %d-%d): validate: %v",
				i, tpl.BuildMin, tpl.BuildMax, err)
		}
	}
}

// TestDefaultTemplates_LayoutsConsistent asserts every shipping
// template's MSVLayout has NodeSize >= every offset we read. Bug
// guard against a future edit that adds an offset without growing
// NodeSize, which would silently truncate the bytes the walker
// projects through the layout.
func TestDefaultTemplates_LayoutsConsistent(t *testing.T) {
	for i, tpl := range builtinTemplates {
		l := tpl.MSVLayout
		maxOff := l.LUIDOffset
		for _, o := range []uint32{
			l.UserNameOffset, l.LogonDomainOffset, l.LogonServerOffset,
			l.LogonTypeOffset, l.LogonTimeOffset, l.SIDOffset,
			l.CredentialsOffset,
		} {
			if o > maxOff {
				maxOff = o
			}
		}
		// CredentialsOffset is a pointer (8 bytes); UNICODE_STRINGs are
		// 16 bytes; LUID is 8 bytes. Demand at least 16 bytes of slack
		// past the largest offset so every read fits.
		need := maxOff + 16
		if l.NodeSize < need {
			t.Errorf("builtinTemplates[%d] (build %d-%d): NodeSize 0x%X < required 0x%X",
				i, tpl.BuildMin, tpl.BuildMax, l.NodeSize, need)
		}
	}
}

// TestDefaultTemplates_NoBuildOverlap asserts no two shipping
// templates cover the same BuildNumber. Overlap would make
// templateFor's "first match wins" rule depend on registration order
// — a footgun for future edits.
func TestDefaultTemplates_NoBuildOverlap(t *testing.T) {
	for i := 0; i < len(builtinTemplates); i++ {
		for j := i + 1; j < len(builtinTemplates); j++ {
			a, b := builtinTemplates[i], builtinTemplates[j]
			if a.BuildMin <= b.BuildMax && b.BuildMin <= a.BuildMax {
				t.Errorf("builtinTemplates[%d] (%d-%d) overlaps [%d] (%d-%d)",
					i, a.BuildMin, a.BuildMax, j, b.BuildMin, b.BuildMax)
			}
		}
	}
}
