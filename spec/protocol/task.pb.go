// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.2
// source: spec/protocol/task.proto

package protocol

import (
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

type SubscriptionTask_SubsctiptionFunc int32

const (
	SubscriptionTask_Unknown     SubscriptionTask_SubsctiptionFunc = 0
	SubscriptionTask_ReportUsage SubscriptionTask_SubsctiptionFunc = 1
	SubscriptionTask_Synchronize SubscriptionTask_SubsctiptionFunc = 2
)

// Enum value maps for SubscriptionTask_SubsctiptionFunc.
var (
	SubscriptionTask_SubsctiptionFunc_name = map[int32]string{
		0: "Unknown",
		1: "ReportUsage",
		2: "Synchronize",
	}
	SubscriptionTask_SubsctiptionFunc_value = map[string]int32{
		"Unknown":     0,
		"ReportUsage": 1,
		"Synchronize": 2,
	}
)

func (x SubscriptionTask_SubsctiptionFunc) Enum() *SubscriptionTask_SubsctiptionFunc {
	p := new(SubscriptionTask_SubsctiptionFunc)
	*p = x
	return p
}

func (x SubscriptionTask_SubsctiptionFunc) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SubscriptionTask_SubsctiptionFunc) Descriptor() protoreflect.EnumDescriptor {
	return file_spec_protocol_task_proto_enumTypes[0].Descriptor()
}

func (SubscriptionTask_SubsctiptionFunc) Type() protoreflect.EnumType {
	return &file_spec_protocol_task_proto_enumTypes[0]
}

func (x SubscriptionTask_SubsctiptionFunc) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SubscriptionTask_SubsctiptionFunc.Descriptor instead.
func (SubscriptionTask_SubsctiptionFunc) EnumDescriptor() ([]byte, []int) {
	return file_spec_protocol_task_proto_rawDescGZIP(), []int{0, 0}
}

type Task_TaskType int32

const (
	Task_Unknown      Task_TaskType = 0
	Task_Subscription Task_TaskType = 1
)

// Enum value maps for Task_TaskType.
var (
	Task_TaskType_name = map[int32]string{
		0: "Unknown",
		1: "Subscription",
	}
	Task_TaskType_value = map[string]int32{
		"Unknown":      0,
		"Subscription": 1,
	}
)

func (x Task_TaskType) Enum() *Task_TaskType {
	p := new(Task_TaskType)
	*p = x
	return p
}

func (x Task_TaskType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Task_TaskType) Descriptor() protoreflect.EnumDescriptor {
	return file_spec_protocol_task_proto_enumTypes[1].Descriptor()
}

func (Task_TaskType) Type() protoreflect.EnumType {
	return &file_spec_protocol_task_proto_enumTypes[1]
}

func (x Task_TaskType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Task_TaskType.Descriptor instead.
func (Task_TaskType) EnumDescriptor() ([]byte, []int) {
	return file_spec_protocol_task_proto_rawDescGZIP(), []int{1, 0}
}

type SubscriptionTask struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Function           SubscriptionTask_SubsctiptionFunc `protobuf:"varint,1,opt,name=Function,proto3,enum=protocol.SubscriptionTask_SubsctiptionFunc" json:"Function,omitempty"`
	SubscriptionItemID string                            `protobuf:"bytes,5,opt,name=SubscriptionItemID,proto3" json:"SubscriptionItemID,omitempty"`
}

func (x *SubscriptionTask) Reset() {
	*x = SubscriptionTask{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spec_protocol_task_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SubscriptionTask) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SubscriptionTask) ProtoMessage() {}

func (x *SubscriptionTask) ProtoReflect() protoreflect.Message {
	mi := &file_spec_protocol_task_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SubscriptionTask.ProtoReflect.Descriptor instead.
func (*SubscriptionTask) Descriptor() ([]byte, []int) {
	return file_spec_protocol_task_proto_rawDescGZIP(), []int{0}
}

func (x *SubscriptionTask) GetFunction() SubscriptionTask_SubsctiptionFunc {
	if x != nil {
		return x.Function
	}
	return SubscriptionTask_Unknown
}

func (x *SubscriptionTask) GetSubscriptionItemID() string {
	if x != nil {
		return x.SubscriptionItemID
	}
	return ""
}

type Task struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Timestamp        *timestamp.Timestamp `protobuf:"bytes,1,opt,name=Timestamp,proto3" json:"Timestamp,omitempty"`
	SubscriptionTask *SubscriptionTask    `protobuf:"bytes,2,opt,name=SubscriptionTask,proto3" json:"SubscriptionTask,omitempty"`
	Type             Task_TaskType        `protobuf:"varint,10,opt,name=Type,proto3,enum=protocol.Task_TaskType" json:"Type,omitempty"`
}

func (x *Task) Reset() {
	*x = Task{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spec_protocol_task_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Task) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Task) ProtoMessage() {}

func (x *Task) ProtoReflect() protoreflect.Message {
	mi := &file_spec_protocol_task_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Task.ProtoReflect.Descriptor instead.
func (*Task) Descriptor() ([]byte, []int) {
	return file_spec_protocol_task_proto_rawDescGZIP(), []int{1}
}

func (x *Task) GetTimestamp() *timestamp.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Task) GetSubscriptionTask() *SubscriptionTask {
	if x != nil {
		return x.SubscriptionTask
	}
	return nil
}

func (x *Task) GetType() Task_TaskType {
	if x != nil {
		return x.Type
	}
	return Task_Unknown
}

var File_spec_protocol_task_proto protoreflect.FileDescriptor

var file_spec_protocol_task_proto_rawDesc = []byte{
	0x0a, 0x18, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f,
	0x74, 0x61, 0x73, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x63, 0x6f, 0x6c, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xce, 0x01, 0x0a, 0x10, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72,
	0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73, 0x6b, 0x12, 0x47, 0x0a, 0x08, 0x46, 0x75,
	0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x2b, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73, 0x6b, 0x2e, 0x53, 0x75, 0x62, 0x73, 0x63, 0x74, 0x69,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x46, 0x75, 0x6e, 0x63, 0x52, 0x08, 0x46, 0x75, 0x6e, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x2e, 0x0a, 0x12, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x49, 0x74, 0x65, 0x6d, 0x49, 0x44, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x12, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x74, 0x65,
	0x6d, 0x49, 0x44, 0x22, 0x41, 0x0a, 0x10, 0x53, 0x75, 0x62, 0x73, 0x63, 0x74, 0x69, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x46, 0x75, 0x6e, 0x63, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x6e, 0x6b, 0x6e, 0x6f,
	0x77, 0x6e, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x55, 0x73,
	0x61, 0x67, 0x65, 0x10, 0x01, 0x12, 0x0f, 0x0a, 0x0b, 0x53, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f,
	0x6e, 0x69, 0x7a, 0x65, 0x10, 0x02, 0x22, 0xe0, 0x01, 0x0a, 0x04, 0x54, 0x61, 0x73, 0x6b, 0x12,
	0x38, 0x0a, 0x09, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x46, 0x0a, 0x10, 0x53, 0x75, 0x62,
	0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73, 0x6b, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x53,
	0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73, 0x6b, 0x52,
	0x10, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73,
	0x6b, 0x12, 0x2b, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x54, 0x61, 0x73, 0x6b, 0x2e,
	0x54, 0x61, 0x73, 0x6b, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x54, 0x79, 0x70, 0x65, 0x22, 0x29,
	0x0a, 0x08, 0x54, 0x61, 0x73, 0x6b, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x6e,
	0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x10, 0x00, 0x12, 0x10, 0x0a, 0x0c, 0x53, 0x75, 0x62, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x10, 0x01, 0x42, 0x29, 0x5a, 0x27, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x7a, 0x6c, 0x6c, 0x6f, 0x76, 0x65, 0x73, 0x75,
	0x6b, 0x69, 0x2f, 0x72, 0x6d, 0x63, 0x2f, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_spec_protocol_task_proto_rawDescOnce sync.Once
	file_spec_protocol_task_proto_rawDescData = file_spec_protocol_task_proto_rawDesc
)

func file_spec_protocol_task_proto_rawDescGZIP() []byte {
	file_spec_protocol_task_proto_rawDescOnce.Do(func() {
		file_spec_protocol_task_proto_rawDescData = protoimpl.X.CompressGZIP(file_spec_protocol_task_proto_rawDescData)
	})
	return file_spec_protocol_task_proto_rawDescData
}

var file_spec_protocol_task_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_spec_protocol_task_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_spec_protocol_task_proto_goTypes = []interface{}{
	(SubscriptionTask_SubsctiptionFunc)(0), // 0: protocol.SubscriptionTask.SubsctiptionFunc
	(Task_TaskType)(0),                     // 1: protocol.Task.TaskType
	(*SubscriptionTask)(nil),               // 2: protocol.SubscriptionTask
	(*Task)(nil),                           // 3: protocol.Task
	(*timestamp.Timestamp)(nil),            // 4: google.protobuf.Timestamp
}
var file_spec_protocol_task_proto_depIdxs = []int32{
	0, // 0: protocol.SubscriptionTask.Function:type_name -> protocol.SubscriptionTask.SubsctiptionFunc
	4, // 1: protocol.Task.Timestamp:type_name -> google.protobuf.Timestamp
	2, // 2: protocol.Task.SubscriptionTask:type_name -> protocol.SubscriptionTask
	1, // 3: protocol.Task.Type:type_name -> protocol.Task.TaskType
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_spec_protocol_task_proto_init() }
func file_spec_protocol_task_proto_init() {
	if File_spec_protocol_task_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_spec_protocol_task_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SubscriptionTask); i {
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
		file_spec_protocol_task_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Task); i {
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
			RawDescriptor: file_spec_protocol_task_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_spec_protocol_task_proto_goTypes,
		DependencyIndexes: file_spec_protocol_task_proto_depIdxs,
		EnumInfos:         file_spec_protocol_task_proto_enumTypes,
		MessageInfos:      file_spec_protocol_task_proto_msgTypes,
	}.Build()
	File_spec_protocol_task_proto = out.File
	file_spec_protocol_task_proto_rawDesc = nil
	file_spec_protocol_task_proto_goTypes = nil
	file_spec_protocol_task_proto_depIdxs = nil
}