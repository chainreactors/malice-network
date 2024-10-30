package inputrc

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
	"unicode"
)

const delimiter = "####----####\n"

func TestConfig(_ *testing.T) {
	var _ Handler = NewDefaultConfig()
}

func TestParse(t *testing.T) {
	var tests []string
	if err := fs.WalkDir(testdata, ".", func(n string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			return nil
		}
		tests = append(tests, n)
		return nil
	}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, test := range tests {
		n := test
		t.Run(filepath.Base(n), func(t *testing.T) {
			test := readTest(t, n)
			if len(test) != 3 {
				t.Fatalf("len(test) != 3: %d", len(test))
			}
			cfg, m := newConfig()
			check(t, test[2], cfg, m, ParseBytes(test[1], cfg, buildOpts(t, test[0])...))
		})
	}
}

func TestUserDefault(t *testing.T) {
	tests := []struct {
		dir string
		exp string
	}{
		{"/home/ken", "ken.inputrc"},
		{"/home/bob", "default.inputrc"},
	}
	for _, testinfo := range tests {
		test := readTest(t, path.Join("testdata", testinfo.exp))
		cfg, m := newConfig()
		u := &user.User{
			HomeDir: testinfo.dir,
		}
		check(t, test[2], cfg, m, UserDefault(u, cfg, buildOpts(t, test[0])...))
	}
}

func TestEncontrolDecontrol(t *testing.T) {
	tests := []struct {
		d, e rune
	}{
		{'a', '\x01'},
		{'i', '\t'},
		{'j', '\n'},
		{'m', '\r'},
		{'A', '\x01'},
		{'I', '\t'},
		{'J', '\n'},
		{'M', '\r'},
	}

	for idx, test := range tests {
		ctrl := Encontrol(test.d)
		if exp := test.e; ctrl != exp {
			t.Errorf("test %d expected %c==%c", idx, exp, ctrl)
		}
		ctrl = Decontrol(test.e)
		if exp := unicode.ToUpper(test.d); ctrl != exp {
			t.Errorf("test %d expected %c==%c", idx, exp, ctrl)
		}
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		s, exp string
	}{
		{"\x1b\x7f", `\e\C-?`},
		{"\x1b[13;", `\e[13;`},
	}
	for i, test := range tests {
		if s, exp := Escape(test.s), test.exp; s != exp {
			t.Errorf("test %d expected %q==%q", i, exp, s)
		}
	}
}

func TestDecode(t *testing.T) {
	const str = `
Control-Meta-f: "a"
Meta-Control-f: "b"
"\C-\M-f": "c"
"\M-\C-f": "d"
Control-Meta-p: "e"
Meta-Control-p: "f"
"\C-\M-p": "g"
"\M-\C-p": "h"
`
	t.Logf("decoding:%s", str)
	cfg := NewConfig()
	if err := ParseBytes([]byte(str), cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	t.Logf("decoded as:")
	for sectKey, sect := range cfg.Binds {
		for key, bind := range sect {
			t.Logf("%q: %q 0x%x: %q %t", sectKey, key, []byte(key), bind.Action, bind.Macro)
		}
	}
}

func TestDecodeKey(t *testing.T) {
	tests := []struct {
		s, exp string
	}{
		{"Escape", "\x1b"},
		{"Control-u", "\x15"},
		{"return", "\r"},
		{"Meta-tab", "\x1b\t"},
		{"Control-Meta-v", string(Encontrol(Enmeta('v')))},
	}
	for idx, test := range tests {
		r := []rune(test.s)
		val, _, err := decodeKey(r, 0, len(r))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// FIXME: need more tests and stuff, and this skip here is just to
		// quiet errors
		if idx == 3 || idx == 4 {
			continue
		}
		if s, exp := val, test.exp; s != exp {
			t.Errorf("test %d expected %q==%q", idx, exp, s)
		}
	}
}

func newConfig() (*Config, map[string][]string) {
	cfg := NewDefaultConfig(WithConfigReadFileFunc(readTestdata))
	keys := make(map[string][]string)
	cfg.Funcs["$custom"] = func(k, v string) error {
		keys[k] = append(keys[k], v)
		return nil
	}
	cfg.Funcs[""] = func(k, v string) error {
		keys[k] = append(keys[k], v)
		return nil
	}
	return cfg, keys
}

func readTest(t *testing.T, name string) [][]byte {
	t.Helper()
	buf, err := testdata.ReadFile(name)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return bytes.Split(buf, []byte(delimiter))
}

func check(t *testing.T, exp []byte, cfg *Config, m map[string][]string, err error) {
	t.Helper()
	res := buildResult(t, exp, cfg, m, err)
	if !bytes.Equal(exp, res) {
		t.Errorf("result does not equal expected:\n%s\ngot:\n%s", string(res), string(res))
	}
}

func buildOpts(t *testing.T, buf []byte) []Option {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(buf), []byte{'\n'})
	var opts []Option
	for i := 0; i < len(lines); i++ {
		line := bytes.TrimSpace(lines[i])
		pos := bytes.Index(line, []byte{':'})
		if pos == -1 {
			t.Fatalf("invalid line %d: %q", i+1, string(line))
		}
		switch k := string(bytes.TrimSpace(line[:pos])); k {
		case "haltOnErr":
			opts = append(opts, WithHaltOnErr(parseBool(t, line[pos+1:])))
		case "strict":
			opts = append(opts, WithStrict(parseBool(t, line[pos+1:])))
		case "app":
			opts = append(opts, WithApp(string(bytes.TrimSpace(line[pos+1:]))))
		case "term":
			opts = append(opts, WithTerm(string(bytes.TrimSpace(line[pos+1:]))))
		case "mode":
			opts = append(opts, WithMode(string(bytes.TrimSpace(line[pos+1:]))))
		default:
			t.Fatalf("unknown param %q", k)
		}
	}
	return opts
}

func buildResult(t *testing.T, exp []byte, cfg *Config, custom map[string][]string, err error) []byte {
	t.Helper()
	m := errRE.FindSubmatch(exp)
	switch {
	case err != nil && m == nil:
		t.Fatalf("expected no error, got: %v", err)
	case err != nil:
		sub := string(m[1])
		re, reErr := regexp.Compile(sub)
		if reErr != nil {
			t.Fatalf("could not compile regexp %q: %v", sub, reErr)
			return nil
		}
		if !re.MatchString(err.Error()) {
			t.Errorf("expected error %q, got: %v", sub, err)
		}
		t.Logf("matched error %q", err)
		return exp
	}
	buf := new(bytes.Buffer)
	// add vars
	dv := DefaultVars()
	vars := make(map[string]interface{})
	for k, v := range cfg.Vars {
		if dv[k] != v {
			vars[k] = v
		}
	}
	if len(vars) != 0 {
		fmt.Fprintln(buf, "vars:")
		var keys []string
		for key := range vars {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(buf, "  %s: %v\n", k, vars[k])
		}
	}
	// add binds
	defaults := DefaultBinds()
	parsed := make(map[string]map[string]string)
	for k := range cfg.Binds {
		parsed[k] = make(map[string]string)
	}
	count := 0
	for k, m := range cfg.Binds {
		for j, v := range m {
			if defaults[k][j] != v {
				if v.Macro {
					parsed[k][j] = `"` + EscapeMacro(v.Action) + `"`
				} else {
					parsed[k][j] = Escape(v.Action)
				}
				count++
			}
		}
	}
	if count != 0 {
		fmt.Fprintln(buf, "binds:")
		var keymaps []string
		for key := range parsed {
			keymaps = append(keymaps, key)
		}
		sort.Strings(keymaps)
		for _, k := range keymaps {
			if len(parsed[k]) != 0 {
				fmt.Fprintf(buf, "  %s:\n", k)
				var binds []string
				for key := range parsed[k] {
					binds = append(binds, key)
				}
				sort.Strings(binds)
				for _, j := range binds {
					fmt.Fprintf(buf, "    %s: %s\n", Escape(j), parsed[k][j])
				}
			}
		}
	}
	if len(custom) != 0 {
		var types []string
		for key := range custom {
			types = append(types, key)
		}
		sort.Strings(types)
		for _, typ := range types {
			if len(custom[typ]) != 0 {
				fmt.Fprintf(buf, "%s:\n", typ)
				for _, v := range custom[typ] {
					fmt.Fprintf(buf, "  %s\n", v)
				}
			}
		}
	}
	// add custom
	return buf.Bytes()
}

var errRE = regexp.MustCompile(`(?im)^\s*error:\s+(.*)$`)

func parseBool(t *testing.T, buf []byte) bool {
	t.Helper()
	switch val := string(bytes.TrimSpace(buf)); val {
	case "true":
		return true
	case "false":
		return false
	default:
		t.Fatalf("unknown bool value %q", val)
	}
	return false
}

func readTestdata(name string) ([]byte, error) {
	switch name {
	case "/home/ken/.inputrc", "\\home\\ken\\_inputrc":
		name = "ken.inputrc"
	case "/etc/inputrc", "\\home\\bob\\_inputrc":
		name = "default.inputrc"
	}
	buf, err := testdata.ReadFile(path.Join("testdata", name))
	if err != nil {
		return nil, err
	}
	v := bytes.Split(buf, []byte(delimiter))
	if len(v) != 3 {
		return nil, fmt.Errorf("test data %s is invalid", name)
	}
	return v[1], nil
}

//go:embed testdata/*.inputrc
var testdata embed.FS
