package schema

type Schema struct {
	AdvancedList *AdvancedList `json:"advanced,omitempty"`
	TypesList    *TypesList    `json:"types"`
}

type AdvancedList struct {
	advanceds []Advanced
}

func (a *AdvancedList) Append(advanced Advanced) {
	a.advanceds = append(a.advanceds, advanced)
}

func (a AdvancedList) AsList() []Advanced {
	return a.advanceds // TODO copy?
}

type Advanced struct {
	name string
	Kind string `json:"kind"` // always "advanced"
}

func NewAdvanced(name string) Advanced {
	a := Advanced{}
	a.name = name
	a.Kind = "advanced"
	return a
}

func (a Advanced) GetName() string {
	return a.name
}

type TypesList struct {
	types []Type
}

func (t *TypesList) Append(typ Type) {
	t.types = append(t.types, typ)
}

func (t TypesList) AsList() []Type {
	return t.types // TODO copy?
}

type Type interface {
	TypeDecl() string
	GetName() string
}

type baseType struct {
	Kind string `json:"kind"`
	name string
}

func (t baseType) GetName() string {
	return t.name
}

type TypeStruct struct {
	baseType
	Fields         StructFieldsList      `json:"fields"`
	Representation *StructRepresentation `json:"representation,omitempty"`
}

func NewTypeStruct(name string, representation *StructRepresentation) TypeStruct {
	ts := TypeStruct{}
	ts.name = name
	ts.Kind = "struct"
	ts.Representation = representation
	return ts
}

type StructFieldsList struct {
	fields []StructField
}

func (sfl *StructFieldsList) Append(sf StructField) {
	sfl.fields = append(sfl.fields, sf)
}

func (sfl StructFieldsList) AsList() []StructField {
	return sfl.fields // TODO copy?
}

func (sfl StructFieldsList) Count() int {
	return len(sfl.fields)
}

type StructField struct {
	name     string
	Type     TypeTerm `json:"type"`
	Nullable bool     `json:"nullable,omitempty"`
	Optional bool     `json:"optional,omitempty"`
}

func NewStructField(name string, nullable bool, optional bool, typ TypeTerm) StructField {
	sf := StructField{}
	sf.name = name
	sf.Nullable = nullable
	sf.Optional = optional
	sf.Type = typ
	return sf
}

func (sf StructField) GetName() string {
	return sf.name
}

type StructRepresentation struct {
	Map         *StructRepresentation_Map        `json:"map,omitempty"`
	Tuple       *StructRepresentation_Tuple      `json:"tuple,omitempty"`
	StringPairs *Representation_StringPairs      `json:"stringpairs,omitempty"`
	StringJoin  *StructRepresentation_StringJoin `json:"stringjoin,omitempty"`
	ListPairs   *Representation_ListPairs        `json:"listpairs,omitempty"`
}

type StructRepresentation_Map struct {
	fieldDetails []StructRepresentation_Map_FieldDetails
}

func (srm *StructRepresentation_Map) AddFieldDetails(fieldDetails StructRepresentation_Map_FieldDetails) {
	srm.fieldDetails = append(srm.fieldDetails, fieldDetails)
}

func (srm StructRepresentation_Map) GetFieldDetails(fieldName string) (StructRepresentation_Map_FieldDetails, bool) {
	for _, fd := range srm.fieldDetails {
		if fd.name == fieldName {
			return fd, true
		}
	}
	return StructRepresentation_Map_FieldDetails{name: fieldName}, false
}

func (srm StructRepresentation_Map) FieldDetailsCount() int {
	return len(srm.fieldDetails)
}

type StructRepresentation_Map_FieldDetails struct {
	name     string
	Implicit interface{} `json:"implicit,omitempty"`
	Rename   string      `json:"rename,omitempty"`
}

func NewStructRepresentation_Map_FieldDetails(name string, implicit interface{}, rename string) StructRepresentation_Map_FieldDetails {
	srmfd := StructRepresentation_Map_FieldDetails{}
	srmfd.name = name
	srmfd.Implicit = implicit
	srmfd.Rename = rename
	return srmfd
}

func (srmfd StructRepresentation_Map_FieldDetails) GetName() string {
	return srmfd.name
}

type StructRepresentation_Tuple struct {
	FieldOrder []string `json:"fieldOrder,omitempty"`
}

// shared by Map and String
type Representation_StringPairs struct {
	InnerDelim string `json:"innerDelim"`
	EntryDelim string `json:"entryDelim"`
}

type StructRepresentation_StringJoin struct {
	FieldOrder []string `json:"fieldOrder,omitempty"`
	Join       string   `json:"join,omitempty"`
}

// shared by Map and String
type Representation_ListPairs struct {
	FieldOrder []string `json:"fieldOrder,omitempty"`
}

type TypeMap struct {
	baseType
	KeyType        string             `json:"keyType"`
	ValueType      TypeTerm           `json:"valueType"`
	ValueNullable  bool               `json:"valueNullable,omitempty"`
	Representation *MapRepresentation `json:"representation,omitempty"`
}

func NewTypeMap(name string, keyType string, valueType TypeTerm, valueNullable bool, representation *MapRepresentation) TypeMap {
	tm := TypeMap{}
	tm.name = name
	tm.Kind = "map"
	tm.KeyType = keyType
	tm.ValueType = valueType
	tm.ValueNullable = valueNullable
	tm.Representation = representation
	return tm
}

type MapRepresentation struct {
	Map         *MapRepresentation_Map      `json:"map,omitempty"`
	StringPairs *Representation_StringPairs `json:"stringpairs,omitempty"`
	ListPairs   *Representation_ListPairs   `json:"listpairs,omitempty"`
	Advanced    *AdvancedDataLayoutName     `json:"advanced,omitempty"`
}

type MapRepresentation_Map struct{}

type TypeList struct {
	baseType
	ValueType      TypeTerm            `json:"valueType,omitempty"`
	ValueNullable  bool                `json:"valueNullable,omitempty"`
	Representation *ListRepresentation `json:"representation,omitempty"`
}

func NewTypeList(name string, valueType TypeTerm, valueNullable bool, representation *ListRepresentation) TypeList {
	tm := TypeList{}
	tm.name = name
	tm.Kind = "list"
	tm.ValueType = valueType
	tm.ValueNullable = valueNullable
	tm.Representation = representation
	return tm
}

type ListRepresentation struct {
	Advanced *AdvancedDataLayoutName `json:"advanced,omitempty"`
}

type TypeEnum struct {
	baseType
	Members        EnumMembersList     `json:"members"`
	Representation *EnumRepresentation `json:"representation,omitempty"`
}

func NewTypeEnum(name string, representation *EnumRepresentation) TypeEnum {
	te := TypeEnum{}
	te.name = name
	te.Kind = "enum"
	te.Representation = representation
	return te
}

type EnumValue string

type EnumMembersList struct {
	members []EnumValue
}

func (eml *EnumMembersList) Append(ev EnumValue) {
	eml.members = append(eml.members, ev)
}

func (eml EnumMembersList) AsList() []EnumValue {
	return eml.members // TODO copy?
}

type EnumRepresentation struct {
	String *EnumRepresentation_String `json:"string,omitempty"`
	Int    *EnumRepresentation_Int    `json:"int,omitempty"`
}

type EnumRepresentation_String struct {
	mappings []enumRepresentation_StringMapping
}

func (ers *EnumRepresentation_String) AddMapping(member EnumValue, value string) {
	ers.mappings = append(ers.mappings, enumRepresentation_StringMapping{member, value})
}

func (ers EnumRepresentation_String) GetMapping(member EnumValue) (string, bool) {
	for _, k := range ers.mappings {
		if k.Member == member {
			return k.Value, true
		}
	}
	return "", false
}

type enumRepresentation_StringMapping struct {
	Member EnumValue
	Value  string
}

type EnumRepresentation_Int struct {
	mappings []enumRepresentation_IntMapping
}

func (eri *EnumRepresentation_Int) AddMapping(member EnumValue, value string) {
	eri.mappings = append(eri.mappings, enumRepresentation_IntMapping{member, value})
}

func (eri EnumRepresentation_Int) GetMapping(member EnumValue) (string, bool) {
	for _, k := range eri.mappings {
		if k.Member == member {
			return k.Value, true
		}
	}
	return "", false
}

type enumRepresentation_IntMapping struct {
	Member EnumValue
	Value  string
}

type TypeUnion struct {
	baseType
	Representation *UnionRepresentation `json:"representation,omitempty"`
}

func NewTypeUnion(name string, representation *UnionRepresentation) TypeUnion {
	tu := TypeUnion{}
	tu.name = name
	tu.Kind = "union"
	tu.Representation = representation
	return tu
}

type UnionRepresentation struct {
	Keyed        *UnionRepresentation_Keyed        `json:"keyed,omitempty"`
	Kinded       *UnionRepresentation_Kinded       `json:"kinded,omitempty"`
	Envelope     *UnionRepresentation_Envelope     `json:"envelope,omitempty"`
	Inline       *UnionRepresentation_Inline       `json:"inline,omitempty"`
	StringPrefix *UnionRepresentation_StringPrefix `json:"stringprefix,omitempty"`
	BytesPrefix  *UnionRepresentation_BytesPrefix  `json:"bytesprefix,omitempty"`
}

type UnionRepresentation_Keyed struct {
	mappings []unionRepresentation_KeyedMapping
}

func (urk *UnionRepresentation_Keyed) AddMapping(key string, typ Type) {
	urk.mappings = append(urk.mappings, unionRepresentation_KeyedMapping{key, typ})
}

func (urk UnionRepresentation_Keyed) GetMapping(key string) (Type, bool) {
	for _, k := range urk.mappings {
		if k.Key == key {
			return k.Typ, true
		}
	}
	return nil, false
}

func (urk UnionRepresentation_Keyed) KeyList() []string {
	kl := make([]string, len(urk.mappings))
	for i, d := range urk.mappings {
		kl[i] = d.Key
	}
	return kl
}

type unionRepresentation_KeyedMapping struct {
	Key string
	Typ Type
}

type RepresentationKind string

type UnionRepresentation_Kinded struct {
	mappings []unionRepresentation_KindedMapping
}

func (urk *UnionRepresentation_Kinded) AddMapping(kind RepresentationKind, typ Type) {
	urk.mappings = append(urk.mappings, unionRepresentation_KindedMapping{kind, typ})
}

func (urk UnionRepresentation_Kinded) GetMapping(kind RepresentationKind) (Type, bool) {
	for _, k := range urk.mappings {
		if k.Kind == kind {
			return k.Typ, true
		}
	}
	return nil, false
}

func (urk UnionRepresentation_Kinded) KindList() []RepresentationKind {
	kl := make([]RepresentationKind, len(urk.mappings))
	for i, d := range urk.mappings {
		kl[i] = d.Kind
	}
	return kl
}

type unionRepresentation_KindedMapping struct {
	Kind RepresentationKind
	Typ  Type
}

type UnionRepresentation_StringPrefix struct {
	mappings []unionRepresentation_StringPrefixMapping
}

func (urb *UnionRepresentation_StringPrefix) AddMapping(typ Type, byts string) {
	urb.mappings = append(urb.mappings, unionRepresentation_StringPrefixMapping{typ, byts})
}

func (urb UnionRepresentation_StringPrefix) GetMapping(byts string) (Type, bool) {
	for _, k := range urb.mappings {
		if k.Byts == byts {
			return k.Typ, true
		}
	}
	return nil, false
}

func (urb UnionRepresentation_StringPrefix) KeyList() []string {
	bl := make([]string, len(urb.mappings))
	for i, d := range urb.mappings {
		bl[i] = d.Byts
	}
	return bl
}

func (urb UnionRepresentation_StringPrefix) TypeList() []Type {
	tl := make([]Type, len(urb.mappings))
	for i, d := range urb.mappings {
		tl[i] = d.Typ
	}
	return tl
}

type unionRepresentation_StringPrefixMapping struct {
	Typ  Type
	Byts string
}

type UnionRepresentation_BytesPrefix struct {
	mappings []unionRepresentation_BytesPrefixMapping
}

func (urb *UnionRepresentation_BytesPrefix) AddMapping(typ Type, byts string) {
	urb.mappings = append(urb.mappings, unionRepresentation_BytesPrefixMapping{typ, byts})
}

func (urb UnionRepresentation_BytesPrefix) GetMapping(byts string) (Type, bool) {
	for _, k := range urb.mappings {
		if k.Byts == byts {
			return k.Typ, true
		}
	}
	return nil, false
}

func (urb UnionRepresentation_BytesPrefix) KeyList() []string {
	bl := make([]string, len(urb.mappings))
	for i, d := range urb.mappings {
		bl[i] = d.Byts
	}
	return bl
}

func (urb UnionRepresentation_BytesPrefix) TypeList() []Type {
	tl := make([]Type, len(urb.mappings))
	for i, d := range urb.mappings {
		tl[i] = d.Typ
	}
	return tl
}

type unionRepresentation_BytesPrefixMapping struct {
	Typ  Type
	Byts string
}

type UnionRepresentation_Envelope struct {
	DiscriminantKey   string                 `json:"discriminantKey"`
	ContentKey        string                 `json:"contentKey"`
	DiscriminantTable UnionDiscriminantTable `json:"discriminantTable"`
}

type UnionDiscriminantTable struct {
	mappings []unionDiscriminantTableMapping
}

func (udt *UnionDiscriminantTable) AddMapping(discriminant string, typ Type) {
	udt.mappings = append(udt.mappings, unionDiscriminantTableMapping{discriminant, typ})
}

func (udt UnionDiscriminantTable) GetMapping(discriminant string) (Type, bool) {
	for _, k := range udt.mappings {
		if k.Discriminant == discriminant {
			return k.Typ, true
		}
	}
	return nil, false
}

func (udt UnionDiscriminantTable) DiscriminantList() []string {
	dl := make([]string, len(udt.mappings))
	for i, d := range udt.mappings {
		dl[i] = d.Discriminant
	}
	return dl
}

type unionDiscriminantTableMapping struct {
	Discriminant string
	Typ          Type
}

type UnionRepresentation_Inline struct {
	DiscriminantKey   string                 `json:"discriminantKey"`
	DiscriminantTable UnionDiscriminantTable `json:"discriminantTable"`
}

type NamedType struct {
	baseType
}

func NewNamedType(name string) NamedType {
	nt := NamedType{}
	nt.name = name
	// no kind .. what will it do?
	return nt
}

func NewSimpleType(name, kind string) Type {
	switch kind {
	case "bool":
		return NewTypeBool(name)
	case "bytes":
		return NewTypeBytes(name, nil)
	case "float":
		return NewTypeFloat(name)
	case "int":
		return NewTypeInt(name)
	case "link":
		return NewTypeLink(name, nil) // TODO: drop support for bare "link"?
	case "string":
		return NewTypeString(name)
	case "null":
		return NewTypeNull(name)
	}
	return nil
}

type TypeTerm interface{}

type TypeBool struct {
	baseType
}

func NewTypeBool(name string) TypeBool {
	tb := TypeBool{}
	tb.Kind = "bool"
	tb.name = name
	return tb
}

type TypeString struct {
	baseType
}

func NewTypeString(name string) TypeString {
	ts := TypeString{}
	ts.Kind = "string"
	ts.name = name
	return ts
}

type TypeNull struct {
	baseType
}

func NewTypeNull(name string) TypeNull {
	tn := TypeNull{}
	tn.Kind = "null"
	tn.name = name
	return tn
}

type TypeBytes struct {
	baseType
	Representation *BytesRepresentation `json:"representation,omitempty"`
}

func NewTypeBytes(name string, representation *BytesRepresentation) TypeBytes {
	tb := TypeBytes{}
	tb.name = name
	tb.Kind = "bytes"
	tb.Representation = representation
	return tb
}

type BytesRepresentation struct {
	Advanced *AdvancedDataLayoutName `json:"advanced,omitempty"`
}

type AdvancedDataLayoutName string

type TypeInt struct {
	baseType
}

func NewTypeInt(name string) TypeInt {
	ti := TypeInt{}
	ti.Kind = "int"
	ti.name = name
	return ti
}

type TypeFloat struct {
	baseType
}

func NewTypeFloat(name string) TypeFloat {
	tf := TypeFloat{}
	tf.Kind = "float"
	tf.name = name
	return tf
}

type TypeLink struct {
	baseType
	ExpectedType TypeTerm `json:"expectedType,omitempty"`
}

func NewTypeLink(name string, expectedType TypeTerm) TypeLink {
	tl := TypeLink{}
	tl.name = name
	tl.Kind = "link"
	if expectedType != NewNamedType("Any") {
		tl.ExpectedType = expectedType
	}
	return tl
}

type TypeCopy struct {
	baseType
	FromType string `json:"fromType"`
}

func NewTypeCopy(name string, fromType string) TypeCopy {
	tc := TypeCopy{}
	tc.name = name
	tc.Kind = "copy"
	tc.FromType = fromType
	return tc
}
