package intermediate

import (
	"encoding/json"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// 将 Protobuf 结构体转换为 map[string]interface{}
func ProtobufToMap(pb proto.Message) (map[string]interface{}, error) {
	// 使用 protojson 序列化器将 Protobuf 结构体转换为 JSON 字符串
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,  // 使用 Protobuf 字段名称
		EmitUnpopulated: false, // 输出未填充的字段
	}
	jsonData, err := marshaler.Marshal(pb)
	if err != nil {
		return nil, err
	}

	// 创建一个 map 来接收反序列化后的数据
	var resultMap map[string]interface{}

	// 将 JSON 字符串反序列化为 map[string]interface{}
	if err := json.Unmarshal(jsonData, &resultMap); err != nil {
		return nil, err
	}

	return resultMap, nil
}
