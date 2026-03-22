package models

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/corpix/uarand"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"strconv"
	"time"
)

// Pipeline
type Pipeline struct {
	ID                           uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt                    time.Time `gorm:"->;<-:create;"`
	ListenerId                   string    `gorm:"type:string;uniqueIndex:idx_pipelines_listener_name"`
	Name                         string    `gorm:"type:string;unique;uniqueIndex:idx_pipelines_listener_name"`
	IP                           string    `gorm:"type:string;default:''"`
	Host                         string    `config:"host"`
	Port                         uint32    `config:"port"`
	Type                         string    `gorm:"type:string;"`
	Enable                       bool      `gorm:"type:boolean;"`
	ParamsData                   string    `gorm:"column:params"`
	CertName                     string    `gorm:"type:string;"`
	*implanttypes.PipelineParams `gorm:"-"`
}

func pipelineParamsFromProto(pipeline *clientpb.Pipeline) *implanttypes.PipelineParams {
	return &implanttypes.PipelineParams{
		Parser:     pipeline.Parser,
		Tls:        implanttypes.FromTls(pipeline.Tls),
		Encryption: implanttypes.FromEncryptions(pipeline.Encryption),
		Secure:     implanttypes.FromSecure(pipeline.Secure),
	}
}

func httpPipelineParams(params string, pipeline *clientpb.Pipeline) *implanttypes.PipelineParams {
	merged := pipelineParamsFromProto(pipeline)
	httpParams, err := implanttypes.UnmarshalPipelineParams(params)
	if err != nil || httpParams == nil {
		return merged
	}
	merged.Headers = httpParams.Headers
	merged.ErrorPage = httpParams.ErrorPage
	merged.BodyPrefix = httpParams.BodyPrefix
	merged.BodySuffix = httpParams.BodySuffix
	return merged
}

func customPipelineParams(params string, pipeline *clientpb.Pipeline) *implanttypes.PipelineParams {
	merged := pipelineParamsFromProto(pipeline)
	customParams, err := implanttypes.UnmarshalPipelineParams(params)
	if err != nil || customParams == nil {
		merged.RawCustomParams = params
		return merged
	}
	customParams.Parser = merged.Parser
	customParams.Tls = merged.Tls
	customParams.Encryption = merged.Encryption
	customParams.Secure = merged.Secure
	// Preserve original custom params JSON for non-built-in pipelines (e.g. webshell)
	customParams.RawCustomParams = params
	return customParams
}

func (pipeline *Pipeline) httpParamsString() string {
	if pipeline == nil || pipeline.PipelineParams == nil {
		return ""
	}
	return (&implanttypes.PipelineParams{
		Headers:    pipeline.Headers,
		ErrorPage:  pipeline.ErrorPage,
		BodyPrefix: pipeline.BodyPrefix,
		BodySuffix: pipeline.BodySuffix,
	}).String()
}

func (pipeline *Pipeline) ToProtobuf() *clientpb.Pipeline {
	if pipeline == nil {
		return nil
	}
	switch pipeline.Type {
	case consts.TCPPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Ip:         pipeline.IP,
			Type:       consts.TCPPipeline,
			CertName:   pipeline.CertName,
			Body: &clientpb.Pipeline_Tcp{
				Tcp: &clientpb.TCPPipeline{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
					Host:       pipeline.Host,
					Port:       uint32(pipeline.Port),
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
			Secure:     pipeline.Secure.ToProtobuf(),
		}
	case consts.HTTPPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Ip:         pipeline.IP,
			Type:       consts.HTTPPipeline,
			CertName:   pipeline.CertName,
			Body: &clientpb.Pipeline_Http{
				Http: &clientpb.HTTPPipeline{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
					Host:       pipeline.Host,
					Port:       uint32(pipeline.Port),
					Params:     pipeline.httpParamsString(),
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
			Secure:     pipeline.Secure.ToProtobuf(),
		}
	case consts.BindPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Ip:         pipeline.IP,
			CertName:   pipeline.CertName,
			Type:       consts.BindPipeline,
			Body: &clientpb.Pipeline_Bind{
				Bind: &clientpb.BindPipeline{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
			Secure:     pipeline.Secure.ToProtobuf(),
		}
	case consts.WebsitePipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Ip:         pipeline.IP,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			CertName:   pipeline.CertName,
			Type:       consts.WebsitePipeline,
			Body: &clientpb.Pipeline_Web{
				Web: &clientpb.Website{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
					Root:       pipeline.WebPath,
					Port:       uint32(pipeline.Port),
					Contents:   make(map[string]*clientpb.WebContent),
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
			Secure:     pipeline.Secure.ToProtobuf(),
		}
	case consts.RemPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Type:       consts.RemPipeline,
			Ip:         pipeline.IP,
			CertName:   pipeline.CertName,
			Body: &clientpb.Pipeline_Rem{
				Rem: &clientpb.REM{
					Name:      pipeline.Name,
					Host:      pipeline.Host,
					Port:      pipeline.Port,
					Link:      pipeline.PipelineParams.Link,
					Subscribe: pipeline.PipelineParams.Subscribe,
					Console:   pipeline.Console,
					Agents:    pipeline.Agents,
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
			Secure:     pipeline.Secure.ToProtobuf(),
		}
	default:
		// All non-built-in types (custom/externally-managed pipelines).
		// Prefer the preserved raw custom params over re-marshaling PipelineParams,
		// because PipelineParams may not have fields for custom keys (e.g. suo5_url).
		params := ""
		if pipeline.PipelineParams != nil && pipeline.PipelineParams.RawCustomParams != "" {
			params = pipeline.PipelineParams.RawCustomParams
		} else if pipeline.PipelineParams != nil {
			data, _ := json.Marshal(pipeline.PipelineParams)
			params = string(data)
		}
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Ip:         pipeline.IP,
			Type:       pipeline.Type,
			CertName:   pipeline.CertName,
			Body: &clientpb.Pipeline_Custom{
				Custom: &clientpb.CustomPipeline{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
					Host:       pipeline.Host,
					Port:       pipeline.Port,
					Params:     params,
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
			Secure:     pipeline.Secure.ToProtobuf(),
		}
	}
}
func (pipeline *Pipeline) Address() string {
	return fmt.Sprintf("%s:%d", pipeline.IP, pipeline.Port)
}
func (pipeline *Pipeline) GetUrl() string {
	var schema string
	switch pipeline.Type {
	case consts.HTTPPipeline:
		if pipeline.Tls != nil && pipeline.Tls.Enable {
			schema = "https://"
		} else {
			schema = "http://"
		}
	case consts.TCPPipeline:
		if pipeline.Tls != nil && pipeline.Tls.Enable {
			schema = "tcp+tls://"
		} else {
			schema = "tcp://"
		}
	default:
		schema = ""
	}
	return fmt.Sprintf("%s%s:%d", schema, pipeline.IP, pipeline.Port)
}

// BeforeCreate - GORM hook
func (pipeline *Pipeline) BeforeCreate(tx *gorm.DB) (err error) {
	pipeline.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	pipeline.CreatedAt = time.Now()
	return nil
}

// BeforeSave GORM 钩子 - 保存前将 Params 序列化
func (pipeline *Pipeline) BeforeSave(tx *gorm.DB) error {
	if pipeline.PipelineParams != nil {
		data, err := json.Marshal(pipeline.PipelineParams)
		if err != nil {
			return err
		}
		pipeline.ParamsData = string(data)
	}
	return nil
}

// AfterFind GORM 钩子 - 查询后反序列化
func (pipeline *Pipeline) AfterFind(tx *gorm.DB) error {
	if pipeline.ParamsData == "" {
		return nil
	}
	var params implanttypes.PipelineParams
	if err := json.Unmarshal([]byte(pipeline.ParamsData), &params); err != nil {
		return err
	}
	pipeline.PipelineParams = &params
	return nil
}

func FromPipelinePb(pipeline *clientpb.Pipeline) *Pipeline {
	switch body := pipeline.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		return &Pipeline{
			ListenerId:     pipeline.ListenerId,
			Name:           pipeline.Name,
			Enable:         pipeline.Enable,
			Host:           body.Tcp.Host,
			IP:             pipeline.Ip,
			Port:           body.Tcp.Port,
			Type:           consts.TCPPipeline,
			CertName:       pipeline.CertName,
			PipelineParams: pipelineParamsFromProto(pipeline),
		}
	case *clientpb.Pipeline_Http:
		return &Pipeline{
			ListenerId:     pipeline.ListenerId,
			Name:           pipeline.Name,
			Enable:         pipeline.Enable,
			Host:           body.Http.Host,
			IP:             pipeline.Ip,
			Port:           body.Http.Port,
			Type:           consts.HTTPPipeline,
			CertName:       pipeline.CertName,
			PipelineParams: httpPipelineParams(body.Http.Params, pipeline),
		}
	case *clientpb.Pipeline_Bind:
		return &Pipeline{
			ListenerId:     pipeline.ListenerId,
			Name:           pipeline.Name,
			Enable:         pipeline.Enable,
			IP:             pipeline.Ip,
			Type:           consts.BindPipeline,
			CertName:       pipeline.CertName,
			PipelineParams: pipelineParamsFromProto(pipeline),
		}
	case *clientpb.Pipeline_Rem:
		return &Pipeline{
			ListenerId: pipeline.ListenerId,
			Name:       pipeline.Name,
			Enable:     pipeline.Enable,
			Type:       consts.RemPipeline,
			Host:       body.Rem.Host,
			Port:       body.Rem.Port,
			IP:         pipeline.Ip,
			CertName:   pipeline.CertName,
			PipelineParams: &implanttypes.PipelineParams{
				Parser:     pipeline.Parser,
				Tls:        implanttypes.FromTls(pipeline.Tls),
				Encryption: implanttypes.FromEncryptions(pipeline.Encryption),
				Secure:     implanttypes.FromSecure(pipeline.Secure),
				Link:       body.Rem.Link,
				Subscribe:  body.Rem.Subscribe,
				Console:    body.Rem.Console,
				Agents:     body.Rem.Agents,
			},
		}
	case *clientpb.Pipeline_Web:
		return &Pipeline{
			ListenerId: pipeline.ListenerId,
			Name:       pipeline.Name,
			Enable:     pipeline.Enable,
			IP:         pipeline.Ip,
			Port:       body.Web.Port,
			CertName:   pipeline.CertName,
			Type:       consts.WebsitePipeline,
			PipelineParams: &implanttypes.PipelineParams{
				Parser:     pipeline.Parser,
				Tls:        implanttypes.FromTls(pipeline.Tls),
				Encryption: implanttypes.FromEncryptions(pipeline.Encryption),
				Secure:     implanttypes.FromSecure(pipeline.Secure),
				WebPath:    body.Web.Root,
			},
		}

	case *clientpb.Pipeline_Custom:
		return &Pipeline{
			ListenerId:     pipeline.ListenerId,
			Name:           pipeline.Name,
			Enable:         pipeline.Enable,
			Host:           body.Custom.Host,
			IP:             pipeline.Ip,
			Port:           body.Custom.Port,
			Type:           pipeline.Type,
			CertName:       pipeline.CertName,
			PipelineParams: customPipelineParams(body.Custom.Params, pipeline),
		}
	default:
		// Fallback for nil body or future unknown types.
		typ := pipeline.Type
		if typ == "" {
			typ = consts.CustomPipeline
		}
		return &Pipeline{
			ListenerId:     pipeline.ListenerId,
			Name:           pipeline.Name,
			Enable:         pipeline.Enable,
			IP:             pipeline.Ip,
			Type:           typ,
			CertName:       pipeline.CertName,
			PipelineParams: pipelineParamsFromProto(pipeline),
		}
	}
}

func (pipeline *Pipeline) ToProfile(backend *Pipeline) (implanttypes.ProfileConfig, error) {
	switch pipeline.Type {
	case consts.TCPPipeline:
		return pipeline.DefaultTCPProfile(), nil
	case consts.HTTPPipeline:
		return pipeline.DefaultHTTPProfile(), nil
	case consts.RemPipeline:
		return pipeline.DefaultRemProfile(backend), nil
	default:
		return implanttypes.ProfileConfig{}, fmt.Errorf("'%s' pipeline is not support.", pipeline.Type)
	}
}

func (pipeline *Pipeline) DefaultTCPProfile() implanttypes.ProfileConfig {
	pipelineProtobuf := pipeline.ToProtobuf()
	pipelineProfile := implanttypes.ProfileConfig{}
	pipelineProfile.SetDefaults()
	target := implanttypes.Target{}
	target.Address = pipelineProtobuf.Ip + ":" + strconv.Itoa(int(pipelineProtobuf.GetTcp().Port))
	target.TCP = &implanttypes.TCPProfile{}
	// beacon
	pipelineProfile.Basic.Targets = append(pipelineProfile.Basic.Targets, target)
	// pulse
	pipelineProfile.Pulse.Target = pipelineProtobuf.Ip + ":" + strconv.Itoa(int(pipelineProtobuf.GetTcp().Port))
	pipelineProfile.Pulse.Protocol = consts.TCPPipeline
	// enc
	for _, encryption := range pipelineProtobuf.Encryption {
		// todo dynamic type key
		if encryption.Type == consts.CryptorAES {
			pipelineProfile.Basic.Encryption = consts.CryptorAES
			pipelineProfile.Basic.Key = encryption.Key
		} else if encryption.Type == consts.CryptorXOR {
			pipelineProfile.Pulse.Encryption = consts.CryptorXOR
			pipelineProfile.Pulse.Key = encryption.Key
		}
	}
	return pipelineProfile
}

func (pipeline *Pipeline) DefaultHTTPProfile() implanttypes.ProfileConfig {
	pipelineProtobuf := pipeline.ToProtobuf()
	pipelineProfile := implanttypes.ProfileConfig{}
	pipelineProfile.SetDefaults()
	target := implanttypes.Target{}
	target.Address = pipelineProtobuf.Ip + ":" + strconv.Itoa(int(pipelineProtobuf.GetHttp().Port))
	target.Http = &implanttypes.HttpProfile{
		Method:  "POST",
		Path:    "/",
		Version: "1.1",
		Headers: map[string]string{
			"User-Agent":   uarand.GetRandom(),
			"Content-Type": "application/octet-stream",
		},
	}
	if pipelineProtobuf.Tls != nil && pipelineProtobuf.Tls.Enable {
		target.TLS = &implanttypes.TLSProfile{
			Enable:           true,
			SNI:              pipelineProtobuf.Ip,
			SkipVerification: true,
		}
	}
	// beacon
	pipelineProfile.Basic.Targets = append(pipelineProfile.Basic.Targets, target)
	pipelineProfile.Pulse.Target = pipelineProtobuf.Ip + ":" + strconv.Itoa(int(pipelineProtobuf.GetHttp().Port))
	pipelineProfile.Pulse.Protocol = consts.HTTPPipeline
	// enc
	for _, encryption := range pipelineProtobuf.Encryption {
		// todo dynamic type key
		if encryption.Type == consts.CryptorAES {
			pipelineProfile.Basic.Encryption = consts.CryptorAES
			pipelineProfile.Basic.Key = encryption.Key
		} else if encryption.Type == consts.CryptorXOR {
			pipelineProfile.Pulse.Encryption = consts.CryptorXOR
			pipelineProfile.Pulse.Key = encryption.Key
		}
	}
	return pipelineProfile
}

func (pipeline *Pipeline) DefaultRemProfile(backend *Pipeline) implanttypes.ProfileConfig {
	pipelineProtobuf := pipeline.ToProtobuf()
	pipelineProfile := implanttypes.ProfileConfig{}
	pipelineProfile.SetDefaults()
	target := implanttypes.Target{}
	backendPB := backend.ToProtobuf()

	target.Address = backendPB.Ip + ":" + strconv.Itoa(int(backendPB.GetTcp().Port))
	target.REM = &implanttypes.REMProfile{}
	target.REM.Link = pipelineProtobuf.GetRem().Link
	pipelineProfile.Implant.Enable3rd = true
	pipelineProfile.Implant.ThirdModules = []string{"rem"}
	// beacon
	pipelineProfile.Basic.Targets = append(pipelineProfile.Basic.Targets, target)
	pipelineProfile.Pulse.Target = backendPB.Ip + ":" + strconv.Itoa(int(backendPB.GetTcp().Port))
	pipelineProfile.Pulse.Protocol = backendPB.Type
	// enc
	for _, encryption := range backendPB.Encryption {
		// todo dynamic type key
		if encryption.Type == consts.CryptorAES {
			pipelineProfile.Basic.Encryption = consts.CryptorAES
			pipelineProfile.Basic.Key = encryption.Key
		} else if encryption.Type == consts.CryptorXOR {
			pipelineProfile.Pulse.Encryption = consts.CryptorXOR
			pipelineProfile.Pulse.Key = encryption.Key
		}
	}
	return pipelineProfile
}
