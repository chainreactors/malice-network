package rpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/protobuf/proto"
)

func TestUpdatePipelineTLSRestartsRunningPipeline(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-running", "old-cert", true)
	seedCertificateFixture(t, "new-cert", "")
	controls := respondToPipelineControls(t, listener, nil, nil)

	server := &Server{}
	updated, err := server.UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-running",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "new-cert",
	})
	if err != nil {
		t.Fatalf("UpdatePipelineTLS failed: %v", err)
	}
	if updated.GetCertName() != "new-cert" || !updated.GetEnable() || !updated.GetTls().GetEnable() {
		t.Fatalf("updated pipeline = %#v, want running pipeline bound to new-cert", updated)
	}

	gotControls := <-controls
	if len(gotControls) != 2 || gotControls[0].GetCtrl() != consts.CtrlPipelineStop || gotControls[1].GetCtrl() != consts.CtrlPipelineStart {
		t.Fatalf("controls = %#v, want stop then start", gotControls)
	}
	if gotControls[1].GetJob().GetPipeline().GetCertName() != "new-cert" {
		t.Fatalf("start cert = %q, want new-cert", gotControls[1].GetJob().GetPipeline().GetCertName())
	}

	persisted, err := db.FindPipelineByListener("pipeline-running", listener.Name)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if persisted.CertName != "new-cert" || !persisted.Enable {
		t.Fatalf("persisted pipeline cert/enable = %q/%v, want new-cert/true", persisted.CertName, persisted.Enable)
	}
	runtime := listener.GetPipeline("pipeline-running")
	if runtime == nil || runtime.GetCertName() != "new-cert" || !runtime.GetEnable() {
		t.Fatalf("runtime pipeline = %#v, want running pipeline bound to new-cert", runtime)
	}
	listed, err := server.ListPipelines(context.Background(), &clientpb.Listener{Id: listener.Name})
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}
	if len(listed.GetPipelines()) != 1 || !listed.GetPipelines()[0].GetTls().GetEnable() || listed.GetPipelines()[0].GetTls().GetCert().GetName() != "new-cert" {
		t.Fatalf("listed pipeline TLS = %#v, want hydrated new-cert", listed.GetPipelines())
	}
}

func TestUpdatePipelineTLSReloadsRunningPipelineWhenCertNameIsUnchanged(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-reload", "shared-cert", true)
	controls := respondToPipelineControls(t, listener, nil, nil)

	_, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-reload",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "shared-cert",
	})
	if err != nil {
		t.Fatalf("UpdatePipelineTLS failed: %v", err)
	}
	got := <-controls
	if len(got) != 2 || got[0].GetCtrl() != consts.CtrlPipelineStop || got[1].GetCtrl() != consts.CtrlPipelineStart {
		t.Fatalf("controls = %#v, want stop then start for same-name reload", got)
	}
}

func TestUpdatePipelineTLSOnlyPersistsStoppedPipeline(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-stopped", "old-cert", false)
	seedCertificateFixture(t, "new-cert", "")

	updated, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-stopped",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "new-cert",
	})
	if err != nil {
		t.Fatalf("UpdatePipelineTLS failed: %v", err)
	}
	if updated.GetCertName() != "new-cert" || updated.GetEnable() {
		t.Fatalf("updated pipeline cert/enable = %q/%v, want new-cert/false", updated.GetCertName(), updated.GetEnable())
	}
	select {
	case ctrl := <-listener.Ctrl:
		t.Fatalf("stopped pipeline unexpectedly sent control: %#v", ctrl)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestUpdatePipelineTLSUnbindsStoppedPipeline(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-unbind", "old-cert", false)

	updated, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-unbind",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_DISABLE,
	})
	if err != nil {
		t.Fatalf("UpdatePipelineTLS failed: %v", err)
	}
	if updated.GetCertName() != "" || updated.GetTls().GetEnable() || updated.GetEnable() {
		t.Fatalf("updated pipeline = %#v, want stopped pipeline with TLS disabled", updated)
	}
	select {
	case ctrl := <-listener.Ctrl:
		t.Fatalf("stopped pipeline unexpectedly sent control: %#v", ctrl)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestUpdatePipelineTLSGeneratesAndSavesCertificateForStoppedPipeline(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-generate", "old-cert", false)

	updated, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId:   listener.Name,
		Name:         "pipeline-generate",
		Mode:         clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT,
		Tls:          &clientpb.TLS{Enable: true},
		SaveCert:     true,
		SaveCertName: "generated-cert",
		CertComment:  "generated for test",
	})
	if err != nil {
		t.Fatalf("UpdatePipelineTLS failed: %v", err)
	}
	if updated.GetCertName() != "generated-cert" || !updated.GetTls().GetEnable() || updated.GetEnable() {
		t.Fatalf("updated pipeline = %#v, want stopped pipeline bound to generated-cert", updated)
	}
	stored, err := db.FindCertificate("generated-cert")
	if err != nil {
		t.Fatalf("FindCertificate failed: %v", err)
	}
	if stored.Comment != "generated for test" {
		t.Fatalf("generated certificate comment = %q", stored.Comment)
	}
	if _, err := tls.X509KeyPair([]byte(stored.CertPEM), []byte(stored.KeyPEM)); err != nil {
		t.Fatalf("generated certificate key pair is invalid: %v", err)
	}
	select {
	case ctrl := <-listener.Ctrl:
		t.Fatalf("stopped pipeline unexpectedly sent control: %#v", ctrl)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestUpdatePipelineTLSRestoresOldBindingWhenRestartFails(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-rollback", "old-cert", true)
	seedCertificateFixture(t, "new-cert", "")
	controls := respondToPipelineControls(t, listener, map[int]error{
		1: fmt.Errorf("new certificate rejected"),
	}, []string{consts.CtrlPipelineStop, consts.CtrlPipelineStart, consts.CtrlPipelineStart})

	_, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-rollback",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "new-cert",
	})
	if err == nil || !strings.Contains(err.Error(), "new certificate rejected") {
		t.Fatalf("UpdatePipelineTLS error = %v, want restart failure", err)
	}

	gotControls := <-controls
	if len(gotControls) != 3 {
		t.Fatalf("control count = %d, want 3", len(gotControls))
	}
	if gotControls[1].GetJob().GetPipeline().GetCertName() != "new-cert" {
		t.Fatalf("failed restart cert = %q, want new-cert", gotControls[1].GetJob().GetPipeline().GetCertName())
	}
	if gotControls[2].GetJob().GetPipeline().GetCertName() != "old-cert" {
		t.Fatalf("rollback restart cert = %q, want old-cert", gotControls[2].GetJob().GetPipeline().GetCertName())
	}

	persisted, findErr := db.FindPipelineByListener("pipeline-rollback", listener.Name)
	if findErr != nil {
		t.Fatalf("FindPipelineByListener failed: %v", findErr)
	}
	if persisted.CertName != "old-cert" || !persisted.Enable {
		t.Fatalf("persisted pipeline cert/enable = %q/%v, want old-cert/true", persisted.CertName, persisted.Enable)
	}
	runtime := listener.GetPipeline("pipeline-rollback")
	if runtime == nil || runtime.GetCertName() != "old-cert" || !runtime.GetEnable() {
		t.Fatalf("runtime pipeline = %#v, want restored old-cert runtime", runtime)
	}
}

func TestUpdatePipelineTLSStopsPartiallyStartedRuntimeBeforeRollback(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-partial-start", "old-cert", true)
	seedCertificateFixture(t, "new-cert", "")
	controlsDone := make(chan []*clientpb.JobCtrl, 1)
	go func() {
		controls := make([]*clientpb.JobCtrl, 0, 4)
		for i := 0; i < 4; i++ {
			ctrl := <-listener.Ctrl
			controls = append(controls, ctrl)
			status := &clientpb.JobStatus{CtrlId: ctrl.GetId(), Status: consts.CtrlStatusSuccess}
			if i == 1 {
				partial := proto.Clone(ctrl.GetJob().GetPipeline()).(*clientpb.Pipeline)
				partial.Enable = true
				core.Jobs.AddPipeline(partial)
				status.Status = consts.CtrlStatusFailed
				status.Error = "start status lost"
			}
			listener.CtrlJob.Store(ctrl.GetId(), status)
		}
		controlsDone <- controls
	}()

	_, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-partial-start",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "new-cert",
	})
	if err == nil || !strings.Contains(err.Error(), "start status lost") {
		t.Fatalf("UpdatePipelineTLS error = %v, want failed new start", err)
	}

	controls := <-controlsDone
	wantControls := []string{consts.CtrlPipelineStop, consts.CtrlPipelineStart, consts.CtrlPipelineStop, consts.CtrlPipelineStart}
	for i, want := range wantControls {
		if controls[i].GetCtrl() != want {
			t.Fatalf("control[%d] = %q, want %q", i, controls[i].GetCtrl(), want)
		}
	}
	if controls[2].GetJob().GetPipeline().GetCertName() != "new-cert" {
		t.Fatalf("cleanup stop cert = %q, want new-cert", controls[2].GetJob().GetPipeline().GetCertName())
	}
	if controls[3].GetJob().GetPipeline().GetCertName() != "old-cert" {
		t.Fatalf("rollback start cert = %q, want old-cert", controls[3].GetJob().GetPipeline().GetCertName())
	}
	runtime := listener.GetPipeline("pipeline-partial-start")
	if runtime == nil || runtime.GetCertName() != "old-cert" || !runtime.GetEnable() {
		t.Fatalf("runtime pipeline = %#v, want restored old-cert runtime", runtime)
	}
}

func TestUpdatePipelineTLSLeavesBindingUntouchedWhenStopFails(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-stop-failure", "old-cert", true)
	seedCertificateFixture(t, "new-cert", "")
	controls := respondToPipelineControls(t, listener, map[int]error{
		0: fmt.Errorf("stop rejected"),
	}, []string{consts.CtrlPipelineStop})

	_, err := (&Server{}).UpdatePipelineTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		ListenerId: listener.Name,
		Name:       "pipeline-stop-failure",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "new-cert",
	})
	if err == nil || !strings.Contains(err.Error(), "stop rejected") {
		t.Fatalf("UpdatePipelineTLS error = %v, want stop failure", err)
	}
	if got := <-controls; len(got) != 1 || got[0].GetCtrl() != consts.CtrlPipelineStop {
		t.Fatalf("controls = %#v, want only failed stop", got)
	}

	persisted, findErr := db.FindPipelineByListener("pipeline-stop-failure", listener.Name)
	if findErr != nil {
		t.Fatalf("FindPipelineByListener failed: %v", findErr)
	}
	if persisted.CertName != "old-cert" || !persisted.Enable {
		t.Fatalf("persisted pipeline cert/enable = %q/%v, want old-cert/true", persisted.CertName, persisted.Enable)
	}
	runtime := listener.GetPipeline("pipeline-stop-failure")
	if runtime == nil || runtime.GetCertName() != "old-cert" || !runtime.GetEnable() {
		t.Fatalf("runtime pipeline = %#v, want untouched old-cert runtime", runtime)
	}
}

func TestStartPipelineCertNameUsesTLSUpdateForRunningPipeline(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-start-compat", "old-cert", true)
	seedCertificateFixture(t, "new-cert", "")
	controls := respondToPipelineControls(t, listener, nil, nil)

	_, err := (&Server{}).StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		ListenerId: listener.Name,
		Name:       "pipeline-start-compat",
		CertName:   "new-cert",
	})
	if err != nil {
		t.Fatalf("StartPipeline failed: %v", err)
	}
	got := <-controls
	if len(got) != 2 || got[0].GetCtrl() != consts.CtrlPipelineStop || got[1].GetCtrl() != consts.CtrlPipelineStart {
		t.Fatalf("controls = %#v, want stop then start", got)
	}
	persisted, err := db.FindPipelineByListener("pipeline-start-compat", listener.Name)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if persisted.CertName != "new-cert" || !persisted.Enable {
		t.Fatalf("persisted pipeline cert/enable = %q/%v, want new-cert/true", persisted.CertName, persisted.Enable)
	}
}

func TestStartPipelineCertNameStartsStoppedPipeline(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-start-stopped", "old-cert", false)
	seedCertificateFixture(t, "new-cert", "")
	controls := respondToPipelineControls(t, listener, nil, []string{consts.CtrlPipelineStart})

	_, err := (&Server{}).StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		ListenerId: listener.Name,
		Name:       "pipeline-start-stopped",
		CertName:   "new-cert",
	})
	if err != nil {
		t.Fatalf("StartPipeline failed: %v", err)
	}
	got := <-controls
	if len(got) != 1 || got[0].GetCtrl() != consts.CtrlPipelineStart {
		t.Fatalf("controls = %#v, want one start", got)
	}
	if got[0].GetJob().GetPipeline().GetCertName() != "new-cert" || !got[0].GetJob().GetPipeline().GetTls().GetEnable() {
		t.Fatalf("start pipeline = %#v, want validated new-cert TLS", got[0].GetJob().GetPipeline())
	}
}

func TestStartPipelineAllowsRuntimeSyncBeforeStartStatus(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "pipeline-sync-callback", "old-cert", false)
	server := &Server{}
	callbackResult := make(chan bool, 1)

	go func() {
		ctrl := <-listener.Ctrl
		runtime := proto.Clone(ctrl.GetJob().GetPipeline()).(*clientpb.Pipeline)
		runtime.Enable = true
		syncDone := make(chan error, 1)
		go func() {
			_, err := server.SyncPipeline(context.Background(), runtime)
			syncDone <- err
		}()

		select {
		case syncErr := <-syncDone:
			status := &clientpb.JobStatus{CtrlId: ctrl.GetId(), Status: consts.CtrlStatusSuccess}
			if syncErr != nil {
				status.Status = consts.CtrlStatusFailed
				status.Error = syncErr.Error()
			}
			listener.CtrlJob.Store(ctrl.GetId(), status)
			callbackResult <- false
		case <-time.After(time.Second):
			listener.CtrlJob.Store(ctrl.GetId(), &clientpb.JobStatus{
				CtrlId: ctrl.GetId(),
				Status: consts.CtrlStatusFailed,
				Error:  "runtime SyncPipeline callback blocked",
			})
			<-syncDone
			callbackResult <- true
		}
	}()

	_, err := server.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		ListenerId: listener.Name,
		Name:       "pipeline-sync-callback",
	})
	blocked := <-callbackResult
	if blocked {
		t.Fatal("runtime SyncPipeline callback was blocked by the lifecycle lock")
	}
	if err != nil {
		t.Fatalf("StartPipeline failed: %v", err)
	}
}

func TestUpdateCertificatePreservesCAWhenOmitted(t *testing.T) {
	newRPCTestEnv(t)
	oldCert, oldKey := websitePEMFixture(t)
	newCert, newKey := websitePEMFixture(t)
	if err := db.SaveCertificate(&models.Certificate{
		Name:      "renewed-cert",
		Type:      "imported",
		CertPEM:   oldCert,
		KeyPEM:    oldKey,
		CACertPEM: "existing-ca",
	}); err != nil {
		t.Fatalf("SaveCertificate failed: %v", err)
	}

	_, err := (&Server{}).UpdateCertificate(context.Background(), &clientpb.TLS{
		Cert: &clientpb.Cert{Name: "renewed-cert", Cert: newCert, Key: newKey},
	})
	if err != nil {
		t.Fatalf("UpdateCertificate failed: %v", err)
	}
	updated, err := db.FindCertificate("renewed-cert")
	if err != nil {
		t.Fatalf("FindCertificate failed: %v", err)
	}
	if updated.CACertPEM != "existing-ca" {
		t.Fatalf("CA = %q, want existing-ca", updated.CACertPEM)
	}
	if updated.CertPEM != newCert || updated.KeyPEM != newKey {
		t.Fatal("certificate key pair was not updated")
	}
}

func TestApplyCertificateReportsEveryReference(t *testing.T) {
	newRPCTestEnv(t)
	listener := seedPipelineTLSFixture(t, "http-ref", "shared-cert", false)
	seedPipelineCertificateReference(t, listener.Name, "site-ref", consts.WebsitePipeline, "shared-cert")
	seedPipelineCertificateReference(t, listener.Name, "bind-ref", consts.BindPipeline, "shared-cert")

	result, err := (&Server{}).ApplyCertificate(context.Background(), &clientpb.CertificateApplyRequest{
		CertName: "shared-cert",
	})
	if err != nil {
		t.Fatalf("ApplyCertificate failed: %v", err)
	}
	if result.GetCertName() != "shared-cert" || len(result.GetTargets()) != 3 {
		t.Fatalf("apply result = %#v, want three shared-cert targets", result)
	}

	byName := make(map[string]*clientpb.CertificateApplyTarget)
	for _, target := range result.GetTargets() {
		byName[target.GetPipelineName()] = target
	}
	for _, name := range []string{"http-ref", "site-ref"} {
		if target := byName[name]; target == nil || !target.GetApplied() || target.GetError() != "" {
			t.Fatalf("target %s = %#v, want applied", name, target)
		}
	}
	if target := byName["bind-ref"]; target == nil || target.GetApplied() || !strings.Contains(target.GetError(), "does not support") {
		t.Fatalf("bind target = %#v, want unsupported error", target)
	}
}

func seedPipelineTLSFixture(t testing.TB, name, certName string, running bool) *core.Listener {
	t.Helper()
	seedCertificateFixture(t, certName, "")
	listener := core.NewListener("listener-a", "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listener.Name,
		Enable:     running,
		Type:       consts.HTTPPipeline,
		CertName:   certName,
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{
			Name:       name,
			ListenerId: listener.Name,
			Host:       "127.0.0.1",
			Port:       8443,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	model, err := db.FindPipelineByListener(name, listener.Name)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	model, err = db.UpdatePipelineCert(certName, model)
	if err != nil {
		t.Fatalf("UpdatePipelineCert failed: %v", err)
	}
	if running {
		runtime := model.ToProtobuf()
		runtime.Enable = true
		core.Jobs.AddPipeline(runtime)
	}
	return listener
}

func seedCertificateFixture(t testing.TB, name, ca string) {
	t.Helper()
	certPEM, keyPEM := websitePEMFixture(t)
	if err := db.SaveCertificate(&models.Certificate{
		Name:      name,
		Type:      "imported",
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		CACertPEM: ca,
	}); err != nil {
		t.Fatalf("SaveCertificate(%s) failed: %v", name, err)
	}
}

func seedPipelineCertificateReference(t testing.TB, listenerID, name, pipelineType, certName string) {
	t.Helper()
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Type:       pipelineType,
		CertName:   certName,
	}
	switch pipelineType {
	case consts.WebsitePipeline:
		pipeline.Body = &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       name,
			ListenerId: listenerID,
			Root:       "/",
			Port:       8444,
		}}
	case consts.BindPipeline:
		pipeline.Body = &clientpb.Pipeline_Bind{Bind: &clientpb.BindPipeline{
			Name:       name,
			ListenerId: listenerID,
		}}
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline(%s) failed: %v", name, err)
	}
	model, err := db.FindPipelineByListener(name, listenerID)
	if err != nil {
		t.Fatalf("FindPipelineByListener(%s) failed: %v", name, err)
	}
	if _, err := db.UpdatePipelineCert(certName, model); err != nil {
		t.Fatalf("UpdatePipelineCert(%s) failed: %v", name, err)
	}
}

func respondToPipelineControls(t testing.TB, listener *core.Listener, failures map[int]error, expected []string) <-chan []*clientpb.JobCtrl {
	t.Helper()
	count := len(expected)
	if count == 0 {
		count = 2
	}
	done := make(chan []*clientpb.JobCtrl, 1)
	go func() {
		controls := make([]*clientpb.JobCtrl, 0, count)
		for i := 0; i < count; i++ {
			select {
			case ctrl := <-listener.Ctrl:
				controls = append(controls, ctrl)
				status := &clientpb.JobStatus{CtrlId: ctrl.GetId(), Status: consts.CtrlStatusSuccess}
				if failure := failures[i]; failure != nil {
					status.Status = consts.CtrlStatusFailed
					status.Error = failure.Error()
				}
				listener.CtrlJob.Store(ctrl.GetId(), status)
			case <-time.After(3 * time.Second):
				done <- controls
				return
			}
		}
		done <- controls
	}()
	return done
}
