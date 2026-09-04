// Package scalar, şemadaki özel skaler tipleri taşır.
package scalar

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
)

// UUID, uuid.UUID'yi GraphQL skaleri olarak sunar.
//
// String yerine ayrı bir skaler kullanılıyor çünkü istemci kod üreticileri
// bunu kendi tiplerine eşleyebilir, ve geçersiz bir kimlik resolver'a hiç
// ulaşmadan reddedilir.
type UUID = uuid.UUID

// MarshalUUID, kimliği kanonik metin biçiminde yazar.
func MarshalUUID(id UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.Quote(id.String()))
	})
}

// UnmarshalUUID, gelen değeri kimliğe çevirir.
func UnmarshalUUID(v any) (UUID, error) {
	switch value := v.(type) {
	case string:
		return uuid.Parse(value)
	case []byte:
		return uuid.ParseBytes(value)
	default:
		return uuid.Nil, fmt.Errorf("UUID bir metin olmalı, %T geldi", v)
	}
}

// JSON, şemaya sığmayan serbest yapıları taşır.
//
// Yalnızca denetim kaydı metadata'sı ve LLM test yanıtı gibi biçimi belge
// başına değişen alanlarda kullanılıyor. Şemayı JSON'a kaçarak tanımlamak,
// istemci kod üretimini işe yaramaz hâle getirir.
type JSON map[string]any

// MarshalJSON serializes the free-form map.
func MarshalJSON(m JSON) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		if m == nil {
			_, _ = io.WriteString(w, "null")
			return
		}
		encoded, err := json.Marshal(map[string]any(m))
		if err != nil {
			_, _ = io.WriteString(w, "null")
			return
		}
		_, _ = w.Write(encoded)
	})
}

// UnmarshalJSON reads a free-form map.
func UnmarshalJSON(v any) (JSON, error) {
	switch value := v.(type) {
	case map[string]any:
		return JSON(value), nil
	case string:
		var out map[string]any
		if err := json.Unmarshal([]byte(value), &out); err != nil {
			return nil, err
		}
		return JSON(out), nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("JSON bir nesne olmalı, %T geldi", v)
	}
}
