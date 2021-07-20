package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

func namedElementListMarshalJSON(maybeSlice interface{}) ([]byte, error) {
	v := reflect.ValueOf(maybeSlice)
	if v.Kind() != reflect.Slice {
		panic("attempted to use namedElementListMarshalJSON with a non-slice")
	}

	buffer := bytes.NewBufferString("{")

	for i := 0; i < v.Len(); i++ {
		ele := v.Index(i)
		nm := ele.MethodByName("GetName")
		rv := nm.Call(nil)
		key, err := json.Marshal(rv[0].Interface()) // should be a string
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(ele.Interface()) // element itself
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("%s:%s", key, value))

		if i < v.Len()-1 {
			buffer.WriteString(",")
		}
	}

	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (t TypesList) MarshalJSON() ([]byte, error) {
	b, err := namedElementListMarshalJSON(t.types)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t AdvancedList) MarshalJSON() ([]byte, error) {
	return namedElementListMarshalJSON(t.advanceds)
}

func (t NamedType) MarshalText() ([]byte, error) {
	return []byte(t.name), nil
}

func (t StructFieldsList) MarshalJSON() ([]byte, error) {
	return namedElementListMarshalJSON(t.fields)
}

func (t StructRepresentation_Map) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	if t.FieldDetailsCount() > 0 {
		b, err := namedElementListMarshalJSON(t.fieldDetails)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("\"fields\":%s", string(b)))
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (t EnumMembersList) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")

	l := len(t.members)
	for i := 0; i < l; i++ {
		key, err := json.Marshal(t.members[i])
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("%s:null", key))
		if i < l-1 {
			buffer.WriteString(",")
		}
	}

	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func mappingListToMapMarshalJSON(maybeMapping interface{}, keyField string, valueField string) ([]byte, error) {
	v := reflect.ValueOf(maybeMapping)
	if v.Kind() != reflect.Slice {
		panic("attempted to use mappingListToMapMarshalJSON with a non-slice")
	}

	buffer := bytes.NewBufferString("{")

	for i := 0; i < v.Len(); i++ {
		ele := v.Index(i)
		key, err := json.Marshal(ele.FieldByName(keyField).Interface())
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(ele.FieldByName(valueField).Interface())
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("%s:%s", key, value))
		if i < v.Len()-1 {
			buffer.WriteString(",")
		}
	}

	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (t EnumRepresentation_String) MarshalJSON() ([]byte, error) {
	return mappingListToMapMarshalJSON(t.mappings, "Member", "Value")
}

func (t EnumRepresentation_Int) MarshalJSON() ([]byte, error) {
	return mappingListToMapMarshalJSON(t.mappings, "Member", "Value")
}

func (t UnionRepresentation_Keyed) MarshalJSON() ([]byte, error) {
	return mappingListToMapMarshalJSON(t.mappings, "Key", "Typ")
}

func (t UnionRepresentation_Kinded) MarshalJSON() ([]byte, error) {
	return mappingListToMapMarshalJSON(t.mappings, "Kind", "Typ")
}

func (t UnionRepresentation_BytesPrefix) MarshalJSON() ([]byte, error) {
	return mappingListToMapMarshalJSON(t.mappings, "Byts", "Typ")
}

func (t UnionDiscriminantTable) MarshalJSON() ([]byte, error) {
	return mappingListToMapMarshalJSON(t.mappings, "Discriminant", "Typ")
}
