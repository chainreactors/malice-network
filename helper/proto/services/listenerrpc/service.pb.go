// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.20.3
// source: services/listenerrpc/service.proto

package listenerrpc

import (
	clientpb "github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	implantpb "github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	lispb "github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_services_listenerrpc_service_proto protoreflect.FileDescriptor

var file_services_listenerrpc_service_proto_rawDesc = []byte{
	0x0a, 0x22, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x65,
	0x6e, 0x65, 0x72, 0x72, 0x70, 0x63, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x72, 0x70,
	0x63, 0x1a, 0x1f, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x2f, 0x69, 0x6d, 0x70, 0x6c, 0x61,
	0x6e, 0x74, 0x70, 0x62, 0x2f, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1d, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x2f, 0x6c, 0x69, 0x73,
	0x70, 0x62, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1c, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x70, 0x62, 0x2f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x32,
	0x9e, 0x01, 0x0a, 0x0a, 0x49, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x52, 0x50, 0x43, 0x12, 0x34,
	0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12, 0x16, 0x2e, 0x6c, 0x69, 0x73,
	0x70, 0x62, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x53, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x1a, 0x10, 0x2e, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x45,
	0x6d, 0x70, 0x74, 0x79, 0x12, 0x2f, 0x0a, 0x07, 0x53, 0x79, 0x73, 0x49, 0x6e, 0x66, 0x6f, 0x12,
	0x12, 0x2e, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x53, 0x79, 0x73, 0x49,
	0x6e, 0x66, 0x6f, 0x1a, 0x10, 0x2e, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x29, 0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67, 0x12, 0x0f, 0x2e,
	0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x50, 0x69, 0x6e, 0x67, 0x1a, 0x10,
	0x2e, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x32, 0xbc, 0x05, 0x0a, 0x0b, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x52, 0x50, 0x43,
	0x12, 0x3d, 0x0a, 0x10, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74,
	0x65, 0x6e, 0x65, 0x72, 0x12, 0x17, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x1a, 0x10, 0x2e,
	0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12,
	0x35, 0x0a, 0x10, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x50, 0x69, 0x70, 0x65, 0x6c,
	0x69, 0x6e, 0x65, 0x12, 0x0f, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x50, 0x69, 0x70, 0x65,
	0x6c, 0x69, 0x6e, 0x65, 0x1a, 0x10, 0x2e, 0x69, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x74, 0x70, 0x62,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x3a, 0x0a, 0x0f, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x65, 0x72, 0x57, 0x65, 0x62, 0x73, 0x69, 0x74, 0x65, 0x12, 0x0f, 0x2e, 0x6c, 0x69, 0x73, 0x70,
	0x62, 0x2e, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x1a, 0x16, 0x2e, 0x6c, 0x69, 0x73,
	0x70, 0x62, 0x2e, 0x57, 0x65, 0x62, 0x73, 0x69, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x38, 0x0a, 0x10, 0x53, 0x74, 0x61, 0x72, 0x74, 0x54, 0x63, 0x70, 0x50, 0x69,
	0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x12, 0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x43,
	0x74, 0x72, 0x6c, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x1a, 0x0f, 0x2e, 0x63, 0x6c,
	0x69, 0x65, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x37, 0x0a, 0x0f,
	0x53, 0x74, 0x6f, 0x70, 0x54, 0x63, 0x70, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x12,
	0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x43, 0x74, 0x72, 0x6c, 0x50, 0x69, 0x70, 0x65,
	0x6c, 0x69, 0x6e, 0x65, 0x1a, 0x0f, 0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x70, 0x62, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x39, 0x0a, 0x10, 0x4c, 0x69, 0x73, 0x74, 0x54, 0x63, 0x70,
	0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x12, 0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70,
	0x62, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x1a, 0x10,
	0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73,
	0x12, 0x34, 0x0a, 0x0c, 0x53, 0x74, 0x61, 0x72, 0x74, 0x57, 0x65, 0x62, 0x73, 0x69, 0x74, 0x65,
	0x12, 0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x43, 0x74, 0x72, 0x6c, 0x50, 0x69, 0x70,
	0x65, 0x6c, 0x69, 0x6e, 0x65, 0x1a, 0x0f, 0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x70, 0x62,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x33, 0x0a, 0x0b, 0x53, 0x74, 0x6f, 0x70, 0x57, 0x65,
	0x62, 0x73, 0x69, 0x74, 0x65, 0x12, 0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x43, 0x74,
	0x72, 0x6c, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x1a, 0x0f, 0x2e, 0x63, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x36, 0x0a, 0x0d, 0x55,
	0x70, 0x6c, 0x6f, 0x61, 0x64, 0x57, 0x65, 0x62, 0x73, 0x69, 0x74, 0x65, 0x12, 0x14, 0x2e, 0x6c,
	0x69, 0x73, 0x70, 0x62, 0x2e, 0x57, 0x65, 0x62, 0x73, 0x69, 0x74, 0x65, 0x41, 0x73, 0x73, 0x65,
	0x74, 0x73, 0x1a, 0x0f, 0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x12, 0x34, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x57, 0x65, 0x62, 0x73, 0x69,
	0x74, 0x65, 0x73, 0x12, 0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x4c, 0x69, 0x73, 0x74,
	0x65, 0x6e, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x1a, 0x0f, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62,
	0x2e, 0x57, 0x65, 0x62, 0x73, 0x69, 0x74, 0x65, 0x73, 0x12, 0x3b, 0x0a, 0x0b, 0x53, 0x70, 0x69,
	0x74, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x13, 0x2e, 0x6c, 0x69, 0x73, 0x70, 0x62,
	0x2e, 0x53, 0x70, 0x69, 0x74, 0x65, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x1a, 0x13, 0x2e,
	0x6c, 0x69, 0x73, 0x70, 0x62, 0x2e, 0x53, 0x70, 0x69, 0x74, 0x65, 0x53, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x28, 0x01, 0x30, 0x01, 0x12, 0x37, 0x0a, 0x09, 0x4a, 0x6f, 0x62, 0x53, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x12, 0x13, 0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x70, 0x62, 0x2e, 0x4a,
	0x6f, 0x62, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x1a, 0x11, 0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e,
	0x74, 0x70, 0x62, 0x2e, 0x4a, 0x6f, 0x62, 0x43, 0x74, 0x72, 0x6c, 0x28, 0x01, 0x30, 0x01, 0x42,
	0x4b, 0x5a, 0x49, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x72, 0x65, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x2f, 0x6d, 0x61, 0x6c, 0x69,
	0x63, 0x65, 0x2d, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x2f, 0x68, 0x65, 0x6c, 0x70, 0x65,
	0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x2f, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x72, 0x70, 0x63, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var file_services_listenerrpc_service_proto_goTypes = []interface{}{
	(*lispb.RegisterSession)(nil),  // 0: lispb.RegisterSession
	(*implantpb.SysInfo)(nil),      // 1: implantpb.SysInfo
	(*implantpb.Ping)(nil),         // 2: implantpb.Ping
	(*lispb.RegisterListener)(nil), // 3: lispb.RegisterListener
	(*lispb.Pipeline)(nil),         // 4: lispb.Pipeline
	(*lispb.CtrlPipeline)(nil),     // 5: lispb.CtrlPipeline
	(*lispb.ListenerName)(nil),     // 6: lispb.ListenerName
	(*lispb.WebsiteAssets)(nil),    // 7: lispb.WebsiteAssets
	(*lispb.SpiteSession)(nil),     // 8: lispb.SpiteSession
	(*clientpb.JobStatus)(nil),     // 9: clientpb.JobStatus
	(*implantpb.Empty)(nil),        // 10: implantpb.Empty
	(*lispb.WebsiteResponse)(nil),  // 11: lispb.WebsiteResponse
	(*clientpb.Empty)(nil),         // 12: clientpb.Empty
	(*lispb.Pipelines)(nil),        // 13: lispb.Pipelines
	(*lispb.Websites)(nil),         // 14: lispb.Websites
	(*clientpb.JobCtrl)(nil),       // 15: clientpb.JobCtrl
}
var file_services_listenerrpc_service_proto_depIdxs = []int32{
	0,  // 0: listenerrpc.ImplantRPC.Register:input_type -> lispb.RegisterSession
	1,  // 1: listenerrpc.ImplantRPC.SysInfo:input_type -> implantpb.SysInfo
	2,  // 2: listenerrpc.ImplantRPC.Ping:input_type -> implantpb.Ping
	3,  // 3: listenerrpc.ListenerRPC.RegisterListener:input_type -> lispb.RegisterListener
	4,  // 4: listenerrpc.ListenerRPC.RegisterPipeline:input_type -> lispb.Pipeline
	4,  // 5: listenerrpc.ListenerRPC.RegisterWebsite:input_type -> lispb.Pipeline
	5,  // 6: listenerrpc.ListenerRPC.StartTcpPipeline:input_type -> lispb.CtrlPipeline
	5,  // 7: listenerrpc.ListenerRPC.StopTcpPipeline:input_type -> lispb.CtrlPipeline
	6,  // 8: listenerrpc.ListenerRPC.ListTcpPipelines:input_type -> lispb.ListenerName
	5,  // 9: listenerrpc.ListenerRPC.StartWebsite:input_type -> lispb.CtrlPipeline
	5,  // 10: listenerrpc.ListenerRPC.StopWebsite:input_type -> lispb.CtrlPipeline
	7,  // 11: listenerrpc.ListenerRPC.UploadWebsite:input_type -> lispb.WebsiteAssets
	6,  // 12: listenerrpc.ListenerRPC.ListWebsites:input_type -> lispb.ListenerName
	8,  // 13: listenerrpc.ListenerRPC.SpiteStream:input_type -> lispb.SpiteSession
	9,  // 14: listenerrpc.ListenerRPC.JobStream:input_type -> clientpb.JobStatus
	10, // 15: listenerrpc.ImplantRPC.Register:output_type -> implantpb.Empty
	10, // 16: listenerrpc.ImplantRPC.SysInfo:output_type -> implantpb.Empty
	10, // 17: listenerrpc.ImplantRPC.Ping:output_type -> implantpb.Empty
	10, // 18: listenerrpc.ListenerRPC.RegisterListener:output_type -> implantpb.Empty
	10, // 19: listenerrpc.ListenerRPC.RegisterPipeline:output_type -> implantpb.Empty
	11, // 20: listenerrpc.ListenerRPC.RegisterWebsite:output_type -> lispb.WebsiteResponse
	12, // 21: listenerrpc.ListenerRPC.StartTcpPipeline:output_type -> clientpb.Empty
	12, // 22: listenerrpc.ListenerRPC.StopTcpPipeline:output_type -> clientpb.Empty
	13, // 23: listenerrpc.ListenerRPC.ListTcpPipelines:output_type -> lispb.Pipelines
	12, // 24: listenerrpc.ListenerRPC.StartWebsite:output_type -> clientpb.Empty
	12, // 25: listenerrpc.ListenerRPC.StopWebsite:output_type -> clientpb.Empty
	12, // 26: listenerrpc.ListenerRPC.UploadWebsite:output_type -> clientpb.Empty
	14, // 27: listenerrpc.ListenerRPC.ListWebsites:output_type -> lispb.Websites
	8,  // 28: listenerrpc.ListenerRPC.SpiteStream:output_type -> lispb.SpiteSession
	15, // 29: listenerrpc.ListenerRPC.JobStream:output_type -> clientpb.JobCtrl
	15, // [15:30] is the sub-list for method output_type
	0,  // [0:15] is the sub-list for method input_type
	0,  // [0:0] is the sub-list for extension type_name
	0,  // [0:0] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_services_listenerrpc_service_proto_init() }
func file_services_listenerrpc_service_proto_init() {
	if File_services_listenerrpc_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_services_listenerrpc_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_services_listenerrpc_service_proto_goTypes,
		DependencyIndexes: file_services_listenerrpc_service_proto_depIdxs,
	}.Build()
	File_services_listenerrpc_service_proto = out.File
	file_services_listenerrpc_service_proto_rawDesc = nil
	file_services_listenerrpc_service_proto_goTypes = nil
	file_services_listenerrpc_service_proto_depIdxs = nil
}