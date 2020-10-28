package spec

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/zllovesuki/rmc/spec/protocol"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Parameters defines a key-value type for used in models and protobuf
type Parameters map[string]string

// Scan is used for the sql driver to load from JSON blob into Golang's map data structure
func (p *Parameters) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Failed to unmarshal jsonb value: %s", value)
	}
	if bytes == nil {
		*p = make(Parameters)
		return nil
	}
	return json.Unmarshal(bytes, &p)
}

// Value is used for the sql driver to serialize Parameters into JSON blob to be stored
func (p *Parameters) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// GormDBDataType is gorm package specific, and returning the corresponding column data type depending on the database
func (*Parameters) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

// Clone will return a copy of itself
func (p *Parameters) Clone() Parameters {
	clone := make(Parameters)
	for k, v := range *p {
		clone[k] = v
	}
	return clone
}

// ToProto will convert Parameters into protobuf data type
func (p *Parameters) ToProto() *protocol.Parameters {
	if p == nil {
		return nil
	}
	return &protocol.Parameters{
		Data: p.Clone(),
	}
}

// FromProto will convert protobuf data into Parameters
func (p *Parameters) FromProto(pb *protocol.Parameters) {
	if pb == nil {
		*p = make(Parameters)
		return
	}
	clone := make(map[string]string)
	for k, v := range pb.Data {
		clone[k] = v
	}
	*p = clone
}
