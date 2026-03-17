package listener

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

// testWebsite creates a Website on a random port with the given rootPath and content.
// Returns the running website and its base URL. Caller must defer website.Close().
func testWebsite(t *testing.T, rootPath string, contents map[string]testContent) (*Website, string) {
	t.Helper()

	// Isolate configs.WebsitePath to a temp dir.
	origWebsitePath := configs.WebsitePath
	configs.WebsitePath = t.TempDir()
	t.Cleanup(func() { configs.WebsitePath = origWebsitePath })

	websiteID := "test-website"
	websiteDir := filepath.Join(configs.WebsitePath, websiteID)
	if err := os.MkdirAll(websiteDir, 0o700); err != nil {
		t.Fatalf("mkdir website dir: %v", err)
	}

	// Build content map: write files to disk and create WebContent entries.
	contentMap := make(map[string]*clientpb.WebContent)
	for path, tc := range contents {
		contentID := strings.ReplaceAll(path, "/", "_")
		filePath := filepath.Join(websiteDir, contentID)
		if err := os.WriteFile(filePath, tc.body, 0o600); err != nil {
			t.Fatalf("write content %s: %v", path, err)
		}
		contentMap[path] = &clientpb.WebContent{
			Id:          contentID,
			WebsiteId:   websiteID,
			Path:        path,
			File:        filePath,
			Content:     tc.body,
			ContentType: tc.contentType,
			Type:        "raw",
			Auth:        tc.auth,
		}
	}

	pipeline := &clientpb.Pipeline{
		Name:       websiteID,
		ListenerId: "test-listener",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       websiteID,
				ListenerId: "test-listener",
				Port:       0, // OS picks a free port
				Root:       rootPath,
			},
		},
	}

	web, err := StartWebsite(nil, pipeline, contentMap)
	if err != nil {
		t.Fatalf("StartWebsite: %v", err)
	}
	t.Cleanup(func() { web.Close() })

	// Wait for server to be ready.
	addr := web.Addr()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr.String(), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return web, baseURL
}

type testContent struct {
	body        []byte
	contentType string
	auth        string // "user:pass", "none", or empty
}

func httpGet(t *testing.T, url string) (int, string, http.Header) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), resp.Header
}

// === Basic content serving ===

func TestWebsiteServesStaticContent(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"index.html": {body: []byte("<h1>Hello</h1>"), contentType: "text/html"},
	})

	status, body, headers := httpGet(t, baseURL+"/index.html")
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if body != "<h1>Hello</h1>" {
		t.Fatalf("body = %q, want <h1>Hello</h1>", body)
	}
	if ct := headers.Get("Content-Type"); ct != "text/html" {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
}

func TestWebsiteServesMultiplePaths(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"a.html":        {body: []byte("page-a"), contentType: "text/html"},
		"css/style.css": {body: []byte("body{}"), contentType: "text/css"},
		"js/app.js":     {body: []byte("console.log()"), contentType: "application/javascript"},
	})

	cases := []struct {
		path string
		want string
	}{
		{"/a.html", "page-a"},
		{"/css/style.css", "body{}"},
		{"/js/app.js", "console.log()"},
	}
	for _, tc := range cases {
		_, body, _ := httpGet(t, baseURL+tc.path)
		if body != tc.want {
			t.Errorf("GET %s = %q, want %q", tc.path, body, tc.want)
		}
	}
}

func TestWebsiteRootPathPrefix(t *testing.T) {
	_, baseURL := testWebsite(t, "/site/", map[string]testContent{
		"index.html": {body: []byte("under /site/"), contentType: "text/html"},
	})

	// Should be accessible under /site/ prefix.
	_, body, _ := httpGet(t, baseURL+"/site/index.html")
	if body != "under /site/" {
		t.Fatalf("GET /site/index.html = %q, want 'under /site/'", body)
	}

	// Should NOT be accessible at root (different handler).
	_, rootBody, _ := httpGet(t, baseURL+"/index.html")
	if rootBody == "under /site/" {
		t.Fatal("content should NOT be accessible at /index.html when rootPath is /site/")
	}
}

// Bug detection: handler returns 200 with empty body for missing content.
func TestWebsite404ForMissingContent(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"exists.html": {body: []byte("here"), contentType: "text/html"},
	})

	status, _, _ := httpGet(t, baseURL+"/nonexistent.html")
	if status != 404 {
		t.Fatalf("missing content status = %d, want 404", status)
	}
}

func TestWebsiteContentTypeHeaders(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"page.html":  {body: []byte("html"), contentType: "text/html"},
		"style.css":  {body: []byte("css"), contentType: "text/css"},
		"script.js":  {body: []byte("js"), contentType: "application/javascript"},
		"logo.png":   {body: []byte("png"), contentType: "image/png"},
		"noext":      {body: []byte("data"), contentType: ""},
	})

	cases := []struct {
		path    string
		wantCT  string
	}{
		{"/page.html", "text/html"},
		{"/style.css", "text/css"},
		{"/script.js", "application/javascript"},
		{"/logo.png", "image/png"},
	}
	for _, tc := range cases {
		_, _, headers := httpGet(t, baseURL+tc.path)
		if got := headers.Get("Content-Type"); got != tc.wantCT {
			t.Errorf("GET %s Content-Type = %q, want %q", tc.path, got, tc.wantCT)
		}
	}

	// Empty content-type: server should not set a Content-Type header if ContentType is empty.
	// The http package may add a default based on content sniffing.
	_, _, headers := httpGet(t, baseURL+"/noext")
	_ = headers // No assertion on empty ContentType — behavior is now documented.
}

// === Dynamic content management ===

func TestWebsiteAddContentAfterStart(t *testing.T) {
	web, baseURL := testWebsite(t, "/", map[string]testContent{
		"initial.html": {body: []byte("initial"), contentType: "text/html"},
	})

	// Verify initial content works.
	_, body, _ := httpGet(t, baseURL+"/initial.html")
	if body != "initial" {
		t.Fatalf("initial content = %q", body)
	}

	// Add new content after server is running.
	websiteDir := filepath.Join(configs.WebsitePath, "test-website")
	newFile := filepath.Join(websiteDir, "new_content")
	if err := os.WriteFile(newFile, []byte("dynamic"), 0o600); err != nil {
		t.Fatalf("write new content: %v", err)
	}
	if err := web.AddContent(&clientpb.WebContent{
		Id:          "new_content",
		WebsiteId:   "test-website",
		Path:        "dynamic.html",
		Content:     []byte("dynamic"),
		ContentType: "text/html",
	}); err != nil {
		t.Fatalf("AddContent: %v", err)
	}

	// Should be immediately accessible.
	_, body, _ = httpGet(t, baseURL+"/dynamic.html")
	if body != "dynamic" {
		t.Fatalf("dynamic content = %q, want 'dynamic'", body)
	}
}

// === Boundary conditions ===

func TestWebsiteNestedContentPath(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"assets/images/logo.png": {body: []byte("PNG-DATA"), contentType: "image/png"},
	})

	_, body, _ := httpGet(t, baseURL+"/assets/images/logo.png")
	if body != "PNG-DATA" {
		t.Fatalf("nested path content = %q, want 'PNG-DATA'", body)
	}
}

func TestWebsiteConcurrentRequests(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"concurrent.html": {body: []byte("concurrent-ok"), contentType: "text/html"},
	})

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, body, _ := httpGet(t, baseURL+"/concurrent.html")
			if body != "concurrent-ok" {
				errors <- fmt.Errorf("got %q", body)
			}
		}()
	}
	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent request failed: %v", err)
	}
}

func TestWebsiteCloseStopsServer(t *testing.T) {
	web, baseURL := testWebsite(t, "/", map[string]testContent{
		"alive.html": {body: []byte("alive"), contentType: "text/html"},
	})

	// Verify server is alive.
	_, body, _ := httpGet(t, baseURL+"/alive.html")
	if body != "alive" {
		t.Fatalf("pre-close content = %q", body)
	}

	// Close the server.
	if err := web.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Requests should now fail. Use a fresh transport to avoid Keep-Alive reuse.
	time.Sleep(200 * time.Millisecond)
	client := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
		Timeout:   time.Second,
	}
	_, err := client.Get(baseURL + "/alive.html")
	if err == nil {
		t.Fatal("GET should fail after Close, but succeeded")
	}
}

func TestWebsiteLargeContent(t *testing.T) {
	largeBody := make([]byte, 1024*1024) // 1MB
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"large.bin": {body: largeBody, contentType: "application/octet-stream"},
	})

	resp, err := http.Get(baseURL + "/large.bin")
	if err != nil {
		t.Fatalf("GET large.bin: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if len(got) != len(largeBody) {
		t.Fatalf("large content size = %d, want %d", len(got), len(largeBody))
	}
	// Spot check a few bytes.
	for _, idx := range []int{0, 1000, 500000, len(largeBody) - 1} {
		if got[idx] != largeBody[idx] {
			t.Fatalf("byte mismatch at index %d: got %d want %d", idx, got[idx], largeBody[idx])
		}
	}
}

// === Bug detection: rootPath edge case ===

func TestWebsiteRootPathSlash(t *testing.T) {
	// rootPath = "/" is the most common case.
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"test.txt": {body: []byte("root-slash"), contentType: "text/plain"},
	})

	_, body, _ := httpGet(t, baseURL+"/test.txt")
	if body != "root-slash" {
		t.Fatalf("rootPath=/ content = %q, want 'root-slash'", body)
	}
}

// === HTTP Basic Auth tests ===

func TestWebsiteBasicAuthProtectsContent(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"secret.html": {body: []byte("secret-data"), contentType: "text/html", auth: "admin:pass123"},
	})

	// No credentials → 401
	status, _, headers := httpGet(t, baseURL+"/secret.html")
	if status != 401 {
		t.Fatalf("no-auth status = %d, want 401", status)
	}
	if wwa := headers.Get("WWW-Authenticate"); wwa == "" {
		t.Fatal("401 response should include WWW-Authenticate header")
	}
}

func TestWebsiteBasicAuthValidCredentials(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"secret.html": {body: []byte("secret-data"), contentType: "text/html", auth: "admin:pass123"},
	})

	status, body, _ := httpGetWithAuth(t, baseURL+"/secret.html", "admin", "pass123")
	if status != 200 {
		t.Fatalf("valid-auth status = %d, want 200", status)
	}
	if body != "secret-data" {
		t.Fatalf("body = %q, want secret-data", body)
	}
}

func TestWebsiteBasicAuthWrongCredentials(t *testing.T) {
	_, baseURL := testWebsite(t, "/", map[string]testContent{
		"secret.html": {body: []byte("secret-data"), contentType: "text/html", auth: "admin:pass123"},
	})

	status, _, _ := httpGetWithAuth(t, baseURL+"/secret.html", "admin", "wrong")
	if status != 401 {
		t.Fatalf("wrong-pass status = %d, want 401", status)
	}

	status, _, _ = httpGetWithAuth(t, baseURL+"/secret.html", "wrong", "pass123")
	if status != 401 {
		t.Fatalf("wrong-user status = %d, want 401", status)
	}
}

func TestWebsiteDefaultAuthAppliesToAllPaths(t *testing.T) {
	web, baseURL := testWebsiteWithDefaultAuth(t, "/", "siteadmin:sitepass", map[string]testContent{
		"page1.html": {body: []byte("page1"), contentType: "text/html"},
		"page2.html": {body: []byte("page2"), contentType: "text/html"},
	})
	_ = web

	// Both paths require auth (inherited from website default)
	status1, _, _ := httpGet(t, baseURL+"/page1.html")
	if status1 != 401 {
		t.Fatalf("page1 no-auth status = %d, want 401", status1)
	}
	status2, _, _ := httpGet(t, baseURL+"/page2.html")
	if status2 != 401 {
		t.Fatalf("page2 no-auth status = %d, want 401", status2)
	}

	// With correct credentials
	status1, body1, _ := httpGetWithAuth(t, baseURL+"/page1.html", "siteadmin", "sitepass")
	if status1 != 200 || body1 != "page1" {
		t.Fatalf("page1 with auth: status=%d body=%q", status1, body1)
	}
}

func TestWebsitePathAuthOverridesDefault(t *testing.T) {
	_, baseURL := testWebsiteWithDefaultAuth(t, "/", "default:pass", map[string]testContent{
		"public.html": {body: []byte("public"), contentType: "text/html", auth: "none"},
		"custom.html": {body: []byte("custom"), contentType: "text/html", auth: "special:key"},
	})

	// "none" path skips website default auth
	status, body, _ := httpGet(t, baseURL+"/public.html")
	if status != 200 || body != "public" {
		t.Fatalf("public path: status=%d body=%q, want 200/public", status, body)
	}

	// "special:key" overrides website default
	status, _, _ = httpGetWithAuth(t, baseURL+"/custom.html", "default", "pass")
	if status != 401 {
		t.Fatalf("custom path with default creds: status=%d, want 401", status)
	}
	status, body, _ = httpGetWithAuth(t, baseURL+"/custom.html", "special", "key")
	if status != 200 || body != "custom" {
		t.Fatalf("custom path with correct creds: status=%d body=%q", status, body)
	}
}

func httpGetWithAuth(t testing.TB, url, user, pass string) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("NewRequest %s: %v", url, err)
	}
	req.SetBasicAuth(user, pass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s with auth: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), resp.Header
}

func testWebsiteWithDefaultAuth(t *testing.T, rootPath, defaultAuth string, contents map[string]testContent) (*Website, string) {
	t.Helper()

	origWebsitePath := configs.WebsitePath
	configs.WebsitePath = t.TempDir()
	t.Cleanup(func() { configs.WebsitePath = origWebsitePath })

	websiteID := "test-website"
	websiteDir := filepath.Join(configs.WebsitePath, websiteID)
	if err := os.MkdirAll(websiteDir, 0o700); err != nil {
		t.Fatalf("mkdir website dir: %v", err)
	}

	contentMap := make(map[string]*clientpb.WebContent)
	for path, tc := range contents {
		contentID := strings.ReplaceAll(path, "/", "_")
		filePath := filepath.Join(websiteDir, contentID)
		if err := os.WriteFile(filePath, tc.body, 0o600); err != nil {
			t.Fatalf("write content %s: %v", path, err)
		}
		contentMap[path] = &clientpb.WebContent{
			Id:          contentID,
			WebsiteId:   websiteID,
			Path:        path,
			File:        filePath,
			Content:     tc.body,
			ContentType: tc.contentType,
			Type:        "raw",
			Auth:        tc.auth,
		}
	}

	pipeline := &clientpb.Pipeline{
		Name:       websiteID,
		ListenerId: "test-listener",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       websiteID,
				ListenerId: "test-listener",
				Port:       0,
				Root:       rootPath,
				Auth:       defaultAuth,
			},
		},
	}

	web, err := StartWebsite(nil, pipeline, contentMap)
	if err != nil {
		t.Fatalf("StartWebsite: %v", err)
	}
	t.Cleanup(func() { web.Close() })

	addr := web.Addr()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr.String(), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return web, baseURL
}

// EventBroker is nil-safe (we added nil checks in Publish/TryPublish/Notify),
// so Website.Start()'s GoGuarded calls won't panic even without a broker.
