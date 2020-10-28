package spec

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/zllovesuki/rmc/spec/protocol"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Parameters map[string]string

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

func (p *Parameters) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (*Parameters) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

func (p *Parameters) Clone() Parameters {
	clone := make(Parameters)
	for k, v := range *p {
		clone[k] = v
	}
	return clone
}

func (p *Parameters) ToProto() *protocol.Parameters {
	return &protocol.Parameters{
		Data: *p,
	}
}

func (p *Parameters) FromProto(pb *protocol.Parameters) {
	*p = pb.Data
}
