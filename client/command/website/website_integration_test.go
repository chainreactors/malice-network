//go:build integration

package website

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestNewWebsiteIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	if err := NewWebsite(clientHarness.Console, "site-alpha", "/alpha", "127.0.0.1", 18080, h.ListenerID(), "", nil); err != nil {
		t.Fatalf("NewWebsite failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		website, err := h.GetWebsite("site-alpha")
		return err == nil && website.Enable
	}, "website to be enabled in db")
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipeline, ok := clientHarness.Console.Pipelines["site-alpha"]
		return ok && pipeline.GetWeb() != nil
	}, "website event to populate client pipeline cache")

	website, err := h.GetWebsite("site-alpha")
	if err != nil {
		t.Fatalf("GetWebsite failed: %v", err)
	}
	if website.GetWeb().GetRoot() != "/alpha" {
		t.Fatalf("website root = %q, want %q", website.GetWeb().GetRoot(), "/alpha")
	}
	if !h.JobExists("site-alpha", h.ListenerID()) {
		t.Fatalf("expected website runtime job to exist")
	}

	listCmd := mustWebsiteSubcommand(t, mustWebsiteRoot(t, Commands(clientHarness.Console)), consts.CommandPipelineList)
	parseWebsiteArgs(t, listCmd)
	output := testsupport.CaptureOutput(func() {
		err = ListWebsitesCmd(listCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListWebsitesCmd failed: %v", err)
	}
	if !strings.Contains(output, "site-alpha") || !strings.Contains(output, "/alpha") {
		t.Fatalf("website list output missing expected values:\n%s", output)
	}
}

func TestExistingWebsiteIsLoadedIntoClientStateOnConnect(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-existing", 18079, "/existing")
	site.GetWeb().Contents["/index.html"] = &clientpb.WebContent{
		WebsiteId: "site-existing",
		Path:      "/index.html",
		Content:   []byte("hello"),
		Type:      "raw",
	}
	h.SeedWebsite(t, site, true)

	clientHarness := testsupport.NewClientHarness(t, h)

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipeline, ok := clientHarness.Console.Pipelines["site-existing"]
		if !ok || pipeline.GetWeb() == nil || pipeline.GetWeb().Root != "/existing" {
			return false
		}
		_, ok = pipeline.GetWeb().Contents["/index.html"]
		return ok
	}, "existing website to be present after initial client sync")
}

func TestStartWebsiteStopsExistingWebsiteBeforeRestart(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-restart", 18081, "/restart")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipeline, ok := clientHarness.Console.Pipelines["site-restart"]
		return ok && pipeline.GetWeb() != nil
	}, "existing website to be present in client cache before restart")

	before := len(h.ControlHistory())
	if err := StartWebsite(clientHarness.Console, "site-restart", ""); err != nil {
		t.Fatalf("StartWebsite failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(h.ControlHistory()) >= before+2
	}, "website restart controller history")

	history := h.ControlHistory()[before:]
	if history[0].Ctrl != consts.CtrlWebsiteStop || history[1].Ctrl != consts.CtrlWebsiteStart {
		t.Fatalf("unexpected website ctrl sequence: %s then %s", history[0].Ctrl, history[1].Ctrl)
	}
}

func TestWebContentLifecycleIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-content", 18082, "/content")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	indexPath, err := h.WriteTempFile("index.html", []byte("<h1>hello</h1>"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}
	content, err := AddWebContent(clientHarness.Console, indexPath, "/index.html", "site-content", "raw", "")
	if err != nil {
		t.Fatalf("AddWebContent failed: %v", err)
	}
	if _, err := h.GetWebContent(content.Id); err != nil {
		t.Fatalf("GetWebContent after add failed: %v", err)
	}
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipeline, ok := clientHarness.Console.Pipelines["site-content"]
		if !ok || pipeline.GetWeb() == nil {
			return false
		}
		_, ok = pipeline.GetWeb().Contents["/index.html"]
		return ok
	}, "website content add event to update client cache")

	updatePath, err := h.WriteTempFile("index-updated.html", []byte("<h1>updated</h1>"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}
	updated, err := UpdateWebContent(clientHarness.Console, content.Id, updatePath, "site-content", "text/html")
	if err != nil {
		t.Fatalf("UpdateWebContent failed: %v", err)
	}
	if updated.ContentType != "text/html" {
		t.Fatalf("updated content type = %q, want %q", updated.ContentType, "text/html")
	}
	if updated.Size != uint64(len("<h1>updated</h1>")) {
		t.Fatalf("updated content size = %d, want %d", updated.Size, len("<h1>updated</h1>"))
	}

	body, err := h.ReadWebsiteContent("site-content", updated.Id)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(body) != "<h1>updated</h1>" {
		t.Fatalf("updated website content = %q", string(body))
	}

	listContentCmd := mustWebsiteSubcommand(t, mustWebsiteRoot(t, Commands(clientHarness.Console)), "list-content")
	parseWebsiteArgs(t, listContentCmd, "site-content")
	output := testsupport.CaptureOutput(func() {
		err = ListWebContentCmd(listContentCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListWebContentCmd failed: %v", err)
	}
	if !strings.Contains(output, "site-content") || !strings.Contains(output, "/index.html") {
		t.Fatalf("website content output missing expected values:\n%s", output)
	}

	if _, err := RemoveWebContent(clientHarness.Console, content.Id); err != nil {
		t.Fatalf("RemoveWebContent failed: %v", err)
	}
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := h.GetWebContent(content.Id)
		return err != nil
	}, "website content to be removed")
	if _, err := h.ReadWebsiteContent("site-content", content.Id); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("website content file error = %v, want not exist", err)
	}
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipeline, ok := clientHarness.Console.Pipelines["site-content"]
		if !ok || pipeline.GetWeb() == nil {
			return false
		}
		_, ok = pipeline.GetWeb().Contents["/index.html"]
		return !ok
	}, "website content remove event to update client cache")
}

func TestWebsiteAddCommandSmokeIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-smoke", 18083, "/smoke")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	filePath, err := h.WriteTempFile("smoke.txt", []byte("smoke"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}

	root := mustWebsiteRoot(t, Commands(clientHarness.Console))
	root.SetArgs([]string{"add", filePath, "--website", "site-smoke", "--path", "/smoke.txt"})
	if err := root.Execute(); err != nil {
		t.Fatalf("website add execute failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		contents, err := h.GetWebContents("site-smoke")
		if err != nil {
			return false
		}
		for _, content := range contents {
			if content.Path == "/smoke.txt" {
				return true
			}
		}
		return false
	}, "website smoke content to exist")
}

func TestWebsiteStopCommandHonorsListenerFlag(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-stop-flag", 18084, "/flag")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	root := mustWebsiteRoot(t, Commands(clientHarness.Console))
	root.SetArgs([]string{"stop", "site-stop-flag", "--listener", "missing-listener"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected website stop to fail for an unknown listener")
	}
	if !h.JobExists("site-stop-flag", h.ListenerID()) {
		t.Fatal("website runtime job should remain after failed stop")
	}
}

func TestStartWebsiteRollsBackWhenListenerStartFails(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-start-fail", 18085, "/fail")
	h.SeedWebsite(t, site, false)
	clientHarness := testsupport.NewClientHarness(t, h)
	h.FailNextCtrl(consts.CtrlWebsiteStart, "site-start-fail", errors.New("website bind failed"))

	err := StartWebsite(clientHarness.Console, "site-start-fail", "")
	if err == nil || !strings.Contains(err.Error(), "website bind failed") {
		t.Fatalf("StartWebsite error = %v, want listener failure", err)
	}

	model, getErr := h.GetWebsite("site-start-fail")
	if getErr != nil {
		t.Fatalf("GetWebsite failed: %v", getErr)
	}
	if model.Enable {
		t.Fatal("expected website to remain disabled after failed start")
	}
	if h.JobExists("site-start-fail", h.ListenerID()) {
		t.Fatal("expected no runtime website job after failed start")
	}
}

func TestStopWebsiteIsIdempotentWhenAlreadyStopped(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-already-stopped", 18086, "/stop")
	h.SeedWebsite(t, site, false)
	clientHarness := testsupport.NewClientHarness(t, h)

	if err := StopWebsite(clientHarness.Console, "site-already-stopped"); err != nil {
		t.Fatalf("StopWebsite on stopped site failed: %v", err)
	}

	model, err := h.GetWebsite("site-already-stopped")
	if err != nil {
		t.Fatalf("GetWebsite failed: %v", err)
	}
	if model.Enable {
		t.Fatal("expected stopped website to remain disabled")
	}
}

func TestStopWebsitePropagatesListenerFailure(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-stop-fail", 18087, "/stop-fail")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)
	h.FailNextCtrl(consts.CtrlWebsiteStop, "site-stop-fail", errors.New("website stop failed"))

	err := StopWebsite(clientHarness.Console, "site-stop-fail")
	if err == nil || !strings.Contains(err.Error(), "website stop failed") {
		t.Fatalf("StopWebsite error = %v, want listener failure", err)
	}

	model, getErr := h.GetWebsite("site-stop-fail")
	if getErr != nil {
		t.Fatalf("GetWebsite failed: %v", getErr)
	}
	if !model.Enable {
		t.Fatal("expected website to remain enabled after failed stop")
	}
	if !h.JobExists("site-stop-fail", h.ListenerID()) {
		t.Fatal("expected runtime website job to remain after failed stop")
	}
}

func TestAddWebContentRespectsExplicitContentType(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-type", 18088, "/type")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	filePath, err := h.WriteTempFile("payload.bin", []byte("type-check"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}

	content, err := AddWebContent(clientHarness.Console, filePath, "/payload.bin", "site-type", "text/plain", "")
	if err != nil {
		t.Fatalf("AddWebContent failed: %v", err)
	}
	if content.ContentType != "text/plain" {
		t.Fatalf("content type = %q, want %q", content.ContentType, "text/plain")
	}

	stored, err := h.GetWebContent(content.Id)
	if err != nil {
		t.Fatalf("GetWebContent failed: %v", err)
	}
	if stored.ContentType != "text/plain" {
		t.Fatalf("stored content type = %q, want %q", stored.ContentType, "text/plain")
	}
}

func TestStartWebsiteFailsWhenWebsiteMissing(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	if err := StartWebsite(clientHarness.Console, "missing-site", ""); err == nil {
		t.Fatal("expected StartWebsite to fail for a missing website")
	}
}

func TestStopWebsiteFailsWhenWebsiteMissing(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	if err := StopWebsite(clientHarness.Console, "missing-site"); err == nil {
		t.Fatal("expected StopWebsite to fail for a missing website")
	}
}

func TestAddWebContentFailsWhenWebsiteMissing(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	filePath, err := h.WriteTempFile("missing.txt", []byte("missing"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}

	if _, err := AddWebContent(clientHarness.Console, filePath, "/missing.txt", "missing-site", "raw", ""); err == nil {
		t.Fatal("expected AddWebContent to fail for a missing website")
	}
}

func TestUpdateWebContentFailsWhenContentMissing(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-update-missing", 18089, "/missing")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	filePath, err := h.WriteTempFile("update.txt", []byte("update"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}

	if _, err := UpdateWebContent(clientHarness.Console, "00000000-0000-0000-0000-000000000000", filePath, "site-update-missing", "text/plain"); err == nil {
		t.Fatal("expected UpdateWebContent to fail for a missing content ID")
	}
}

func TestRemoveWebContentFailsWhenContentMissing(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	if _, err := RemoveWebContent(clientHarness.Console, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Fatal("expected RemoveWebContent to fail for a missing content ID")
	}
}

func TestListWebContentCmdReportsEmptyWebsite(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	site := h.NewWebsitePipeline("site-empty", 18090, "/empty")
	h.SeedWebsite(t, site, true)
	clientHarness := testsupport.NewClientHarness(t, h)

	listContentCmd := mustWebsiteSubcommand(t, mustWebsiteRoot(t, Commands(clientHarness.Console)), "list-content")
	parseWebsiteArgs(t, listContentCmd, "site-empty")

	var err error
	output := testsupport.CaptureOutput(func() {
		err = ListWebContentCmd(listContentCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListWebContentCmd failed: %v", err)
	}
	if !strings.Contains(output, "No content found in website site-empty") {
		t.Fatalf("unexpected empty website output:\n%s", output)
	}
}

func mustWebsiteRoot(t testing.TB, commands []*cobra.Command) *cobra.Command {
	t.Helper()

	for _, cmd := range commands {
		if cmd.Name() == consts.CommandWebsite {
			return cmd
		}
	}
	t.Fatalf("website root command not found")
	return nil
}

func mustWebsiteSubcommand(t testing.TB, root *cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range root.Commands() {
		if cmd.Name() == name || strings.Split(cmd.Use, " ")[0] == name {
			return cmd
		}
	}
	t.Fatalf("website subcommand %q not found", name)
	return nil
}

func parseWebsiteArgs(t testing.TB, cmd *cobra.Command, args ...string) {
	t.Helper()

	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
}
