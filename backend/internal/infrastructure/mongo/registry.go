package mongo

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// uuidType, kodek kaydında kullanılan yansıma tipi.
var uuidType = reflect.TypeOf(uuid.UUID{})

// NewRegistry, uuid.UUID'yi BSON binary (subtype 4) olarak kodlayan bir kayıt
// döndürür.
//
// Varsayılan davranış işe yaramaz: uuid.UUID aslında [16]byte'tır ve sürücü
// onu on altı elemanlı bir BSON dizisi olarak yazar. Bu hâliyle belgeler
// mongosh'ta okunamaz, index'ler şişer, ve başka bir dilde yazılmış bir
// istemci aynı belgeyi UUID olarak göremez. Subtype 4, UUID'nin standart
// BSON temsilidir.
func NewRegistry() *bson.Registry {
	reg := bson.NewRegistry()
	reg.RegisterTypeEncoder(uuidType, bson.ValueEncoderFunc(encodeUUID))
	reg.RegisterTypeDecoder(uuidType, bson.ValueDecoderFunc(decodeUUID))
	return reg
}

func encodeUUID(_ bson.EncodeContext, vw bson.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != uuidType {
		return bson.ValueEncoderError{Name: "encodeUUID", Types: []reflect.Type{uuidType}, Received: val}
	}
	id := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(id[:], bson.TypeBinaryUUID)
}

func decodeUUID(_ bson.DecodeContext, vr bson.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != uuidType {
		return bson.ValueDecoderError{Name: "decodeUUID", Types: []reflect.Type{uuidType}, Received: val}
	}

	switch vr.Type() {
	case bson.TypeBinary:
		data, _, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		parsed, err := uuid.FromBytes(data)
		if err != nil {
			return fmt.Errorf("geçersiz UUID binary: %w", err)
		}
		val.Set(reflect.ValueOf(parsed))
		return nil

	case bson.TypeString:
		// Elle yazılmış ya da başka bir araçla eklenmiş belgeler UUID'yi
		// metin olarak taşıyabilir. Okumada kabul ediliyor, yazmada asla
		// üretilmiyor: tek bir kanonik temsil olmalı.
		text, err := vr.ReadString()
		if err != nil {
			return err
		}
		parsed, err := uuid.Parse(text)
		if err != nil {
			return fmt.Errorf("geçersiz UUID metni: %w", err)
		}
		val.Set(reflect.ValueOf(parsed))
		return nil

	case bson.TypeNull:
		if err := vr.ReadNull(); err != nil {
			return err
		}
		val.Set(reflect.ValueOf(uuid.Nil))
		return nil

	default:
		return fmt.Errorf("UUID okunamıyor: beklenmeyen BSON tipi %s", vr.Type())
	}
}
