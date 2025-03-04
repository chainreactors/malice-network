// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.1
// 	protoc        v5.29.3
// source: client/rootpb/root.proto

package rootpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Operator struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name   string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Op     string            `protobuf:"bytes,2,opt,name=op,proto3" json:"op,omitempty"`
	Args   []string          `protobuf:"bytes,4,rep,name=args,proto3" json:"args,omitempty"`
	Params map[string]string `protobuf:"bytes,5,rep,name=params,proto3" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Operator) Reset() {
	*x = Operator{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_rootpb_root_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Operator) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Operator) ProtoMessage() {}

func (x *Operator) ProtoReflect() protoreflect.Message {
	mi := &file_client_rootpb_root_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Operator.ProtoReflect.Descriptor instead.
func (*Operator) Descriptor() ([]byte, []int) {
	return file_client_rootpb_root_proto_rawDescGZIP(), []int{0}
}

func (x *Operator) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Operator) GetOp() string {
	if x != nil {
		return x.Op
	}
	return ""
}

func (x *Operator) GetArgs() []string {
	if x != nil {
		return x.Args
	}
	return nil
}

func (x *Operator) GetParams() map[string]string {
	if x != nil {
		return x.Params
	}
	return nil
}

type Response struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status   int32  `protobuf:"varint,1,opt,name=status,proto3" json:"status,omitempty"`
	Response string `protobuf:"bytes,2,opt,name=response,proto3" json:"response,omitempty"`
	Error    string `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *Response) Reset() {
	*x = Response{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_rootpb_root_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_client_rootpb_root_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_client_rootpb_root_proto_rawDescGZIP(), []int{1}
}

func (x *Response) GetStatus() int32 {
	if x != nil {
		return x.Status
	}
	return 0
}

func (x *Response) GetResponse() string {
	if x != nil {
		return x.Response
	}
	return ""
}

func (x *Response) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

var File_client_rootpb_root_proto protoreflect.FileDescriptor

var file_client_rootpb_root_proto_rawDesc = []byte{
	0x0a, 0x18, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x72, 0x6f, 0x6f, 0x74, 0x70, 0x62, 0x2f,
	0x72, 0x6f, 0x6f, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x72, 0x6f, 0x6f, 0x74,
	0x70, 0x62, 0x22, 0xb3, 0x01, 0x0a, 0x08, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x6f, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x6f, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73, 0x12, 0x34, 0x0a, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d,
	0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x72, 0x6f, 0x6f, 0x74, 0x70, 0x62,
	0x2e, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x1a, 0x39, 0x0a,
	0x0b, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x54, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1a, 0x0a, 0x08,
	0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x42, 0x44,
	0x5a, 0x42, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x72, 0x65, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x2f, 0x6d, 0x61, 0x6c, 0x69, 0x63,
	0x65, 0x2d, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x2f, 0x68, 0x65, 0x6c, 0x70, 0x65, 0x72,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x72, 0x6f,
	0x6f, 0x74, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_client_rootpb_root_proto_rawDescOnce sync.Once
	file_client_rootpb_root_proto_rawDescData = file_client_rootpb_root_proto_rawDesc
)

func file_client_rootpb_root_proto_rawDescGZIP() []byte {
	file_client_rootpb_root_proto_rawDescOnce.Do(func() {
		file_client_rootpb_root_proto_rawDescData = protoimpl.X.CompressGZIP(file_client_rootpb_root_proto_rawDescData)
	})
	return file_client_rootpb_root_proto_rawDescData
}

var file_client_rootpb_root_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_client_rootpb_root_proto_goTypes = []interface{}{
	(*Operator)(nil), // 0: rootpb.Operator
	(*Response)(nil), // 1: rootpb.Response
	nil,              // 2: rootpb.Operator.ParamsEntry
}
var file_client_rootpb_root_proto_depIdxs = []int32{
	2, // 0: rootpb.Operator.params:type_name -> rootpb.Operator.ParamsEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_client_rootpb_root_proto_init() }
func file_client_rootpb_root_proto_init() {
	if File_client_rootpb_root_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_client_rootpb_root_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Operator); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_client_rootpb_root_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_client_rootpb_root_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_client_rootpb_root_proto_goTypes,
		DependencyIndexes: file_client_rootpb_root_proto_depIdxs,
		MessageInfos:      file_client_rootpb_root_proto_msgTypes,
	}.Build()
	File_client_rootpb_root_proto = out.File
	file_client_rootpb_root_proto_rawDesc = nil
	file_client_rootpb_root_proto_goTypes = nil
	file_client_rootpb_root_proto_depIdxs = nil
}
