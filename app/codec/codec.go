package codec

import (
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type UUIDCodec struct{}

// ENCODE UUID → BSON
func (c *UUIDCodec) EncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	u := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(u[:], bsontype.BinaryUUID)
}

// DECODE BSON → UUID
func (c *UUIDCodec) DecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}

	if subtype != bsontype.BinaryUUID {
		return bsoncodec.ValueDecoderError{Name: "UUIDCodec", Types: []reflect.Type{reflect.TypeOf(uuid.UUID{})}}
	}

	u, err := uuid.FromBytes(data)
	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(u))
	return nil
}
