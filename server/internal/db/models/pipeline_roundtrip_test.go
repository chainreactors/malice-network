package models

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestHTTPPipelineRoundTripPreservesParamsAndCommonFields(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "http-1",
		ListenerId: "listener-1",
		Enable:     true,
		Parser:     "auto",
		Ip:         "127.0.0.1",
		Type:       consts.HTTPPipeline,
		CertName:   "cert-a",
		Tls: &clientpb.TLS{
			Enable: true,
			Domain: "example.com",
			CertSubject: &clientpb.CertificateSubject{
				Cn: "subject-cn",
			},
		},
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorAES, Key: "aes-key"},
		},
		Secure: &clientpb.Secure{
			Enable: true,
			ServerKeypair: &clientpb.KeyPair{
				PublicKey:  "spub",
				PrivateKey: "spriv",
			},
			ImplantKeypair: &clientpb.KeyPair{
				PublicKey:  "ipub",
				PrivateKey: "ipriv",
			},
		},
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Host:   "0.0.0.0",
				Port:   8080,
				Params: (&implanttypes.PipelineParams{Headers: map[string][]string{"X-Test": []string{"a", "b"}}, ErrorPage: "err", BodyPrefix: "pre", BodySuffix: "suf"}).String(),
			},
		},
	}

	model := FromPipelinePb(pb)
	if model.Headers["X-Test"][0] != "a" || model.ErrorPage != "err" || model.BodyPrefix != "pre" || model.BodySuffix != "suf" {
		t.Fatalf("http params not preserved in model: %#v", model.PipelineParams)
	}
	if model.Tls == nil || model.Tls.Subject == nil || model.Tls.Subject.CommonName != "subject-cn" {
		t.Fatalf("tls subject not preserved in model: %#v", model.Tls)
	}
	if model.Secure == nil || model.Secure.ServerPublicKey != "spub" || model.Secure.ImplantPrivateKey != "ipriv" {
		t.Fatalf("secure config not preserved in model: %#v", model.Secure)
	}

	roundTrip := model.ToProtobuf()
	params, err := implanttypes.UnmarshalPipelineParams(roundTrip.GetHttp().Params)
	if err != nil {
		t.Fatalf("failed to unmarshal http params: %v", err)
	}
	if params.ErrorPage != "err" || params.BodyPrefix != "pre" || params.BodySuffix != "suf" {
		t.Fatalf("http params lost on protobuf round trip: %#v", params)
	}
	if len(params.Headers["X-Test"]) != 2 {
		t.Fatalf("headers lost on protobuf round trip: %#v", params.Headers)
	}
	if roundTrip.Secure == nil || !roundTrip.Secure.Enable {
		t.Fatalf("secure config lost on protobuf round trip: %#v", roundTrip.Secure)
	}
	if roundTrip.Tls == nil || roundTrip.Tls.CertSubject == nil || roundTrip.Tls.CertSubject.Cn != "subject-cn" {
		t.Fatalf("tls subject lost on protobuf round trip: %#v", roundTrip.Tls)
	}
}

func TestBindPipelineRoundTripPreservesSecure(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "bind-1",
		ListenerId: "listener-1",
		Enable:     true,
		Parser:     consts.ImplantMalefic,
		Ip:         "127.0.0.1",
		Type:       consts.BindPipeline,
		Secure: &clientpb.Secure{
			Enable: true,
			ServerKeypair: &clientpb.KeyPair{
				PublicKey:  "spub",
				PrivateKey: "spriv",
			},
		},
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
	}

	model := FromPipelinePb(pb)
	if model.Secure == nil || !model.Secure.Enable || model.Secure.ServerPrivateKey != "spriv" {
		t.Fatalf("bind secure config not preserved in model: %#v", model.Secure)
	}

	roundTrip := model.ToProtobuf()
	if roundTrip.Secure == nil || !roundTrip.Secure.Enable || roundTrip.Secure.ServerKeypair.PrivateKey != "spriv" {
		t.Fatalf("bind secure config lost on protobuf round trip: %#v", roundTrip.Secure)
	}
}

func TestWebsitePipelineRoundTripPreservesCommonFields(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "web-1",
		ListenerId: "listener-1",
		Enable:     true,
		Parser:     "auto",
		Ip:         "127.0.0.1",
		Type:       consts.WebsitePipeline,
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorXOR, Key: "xor-key"},
		},
		Secure: &clientpb.Secure{
			Enable: true,
		},
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Root: "/static",
				Port: 8081,
			},
		},
	}

	model := FromPipelinePb(pb)
	if model.Parser != "auto" || model.WebPath != "/static" || len(model.Encryption) != 1 || model.Secure == nil || !model.Secure.Enable {
		t.Fatalf("website fields not preserved in model: %#v", model.PipelineParams)
	}

	roundTrip := model.ToProtobuf()
	if roundTrip.Parser != "auto" || len(roundTrip.Encryption) != 1 || roundTrip.Secure == nil || !roundTrip.Secure.Enable {
		t.Fatalf("website common fields lost on protobuf round trip: %#v", roundTrip)
	}
}

func TestCustomPipelineRoundTripPreservesCustomAndCommonParams(t *testing.T) {
	customParams := (&implanttypes.PipelineParams{
		WebPath:    "/custom",
		Headers:    map[string][]string{"X-Custom": []string{"1"}},
		BodyPrefix: "prefix",
	}).String()

	pb := &clientpb.Pipeline{
		Name:       "custom-1",
		ListenerId: "listener-1",
		Enable:     true,
		Parser:     "auto",
		Ip:         "127.0.0.1",
		Type:       "custom",
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorAES, Key: "aes-key"},
		},
		Secure: &clientpb.Secure{
			Enable: true,
		},
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{
				Host:   "127.0.0.1",
				Port:   9000,
				Params: customParams,
			},
		},
	}

	model := FromPipelinePb(pb)
	if model.WebPath != "/custom" || model.BodyPrefix != "prefix" || len(model.Encryption) != 1 || model.Secure == nil || !model.Secure.Enable {
		t.Fatalf("custom params not preserved in model: %#v", model.PipelineParams)
	}

	roundTrip := model.ToProtobuf()
	params, err := implanttypes.UnmarshalPipelineParams(roundTrip.GetCustom().Params)
	if err != nil {
		t.Fatalf("failed to unmarshal custom params: %v", err)
	}
	if params.WebPath != "/custom" || params.BodyPrefix != "prefix" {
		t.Fatalf("custom params lost on protobuf round trip: %#v", params)
	}
	if roundTrip.Secure == nil || !roundTrip.Secure.Enable || len(roundTrip.Encryption) != 1 {
		t.Fatalf("custom common fields lost on protobuf round trip: %#v", roundTrip)
	}
}
