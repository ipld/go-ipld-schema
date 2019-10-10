package schema

import (
	"fmt"
	"strings"
)

type Schema struct {
	AdvancedMap AdvancedMap `json:"advanced,omitempty"`
	TypesMap    TypesMap    `json:"types"`
}

type AdvancedMap map[string]Advanced

type Advanced struct {
	Kind string `json:"kind"` // always "advanced"
}

type TypesMap map[string]Type

type Type interface {
	TypeDecl() string
}

type TypeMap struct {
	KeyType        string             `json:"keyType"`
	Kind           string             `json:"kind"`
	Representation *MapRepresentation `json:"representation,omitempty"`
	ValueNullable  bool               `json:"valueNullable,omitempty"`
	ValueType      TypeTerm           `json:"valueType"`
}

type TypeStruct struct {
	Fields         map[string]*StructField `json:"fields"`
	Kind           string                  `json:"kind"`
	Representation *StructRepresentation   `json:"representation,omitempty"`
}

func (t *TypeStruct) TypeDecl() string {
	term := "struct {\n"

	fieldInfo := make(map[string]SRMFieldDetails)
	if t.Representation != nil && t.Representation.Map != nil {
		fieldInfo = t.Representation.Map.Fields
	}

	for fname, f := range t.Fields {
		term += fmt.Sprintf("\t%s", fname)
		if f.Optional {
			term += " optional"
		}
		if f.Nullable {
			term += " nullable"
		}
		term += " "
		term += TypeTermDecl(f.Type)

		if inf, ok := fieldInfo[fname]; ok && (inf.Rename != "" || inf.Implicit != nil) {
			term += " ("
			if inf.Rename != "" {
				term += fmt.Sprintf("rename \"%s\"", inf.Rename)
			}
			if inf.Implicit != nil {
				if inf.Rename != "" {
					term += " "
				}
				var implicit interface{}
				switch v := inf.Implicit.(type) {
				case string:
					implicit = "\"" + v + "\""
				default:
					implicit = v
				}
				// implicits are always quoted and coerced when parsed
				impl := fmt.Sprintf("%v", implicit)
				if impl[0] != '"' || impl[len(impl)-1] != '"' {
					impl = "\"" + impl + "\""
				}
				term += fmt.Sprintf("implicit %s", impl)
			}
			term += ")"
		}
		term += "\n"
	}
	term += "}"
	if t.Representation != nil {
		// ignore t.Representation.Map
		if t.Representation.Tuple != nil {
			term += " representation tuple"
			if t.Representation.Tuple.FieldOrder != nil {
				term += fmt.Sprintf(" {\n\tfieldOrder [\"%s\"]\n}", strings.Join(t.Representation.Tuple.FieldOrder, "\", \""))
			}
		} else if t.Representation.StringPairs != nil {
			term += stringPairsRepresentationDecl(t.Representation.StringPairs)
		} else if t.Representation.StringJoin != nil {
			term += fmt.Sprintf(" representation stringjoin {\n\tjoin \"%s\"\n", t.Representation.StringJoin.Join)
			if t.Representation.StringJoin.FieldOrder != nil {
				term += fmt.Sprintf("\tfieldOrder [\"%s\"]\n", strings.Join(t.Representation.StringJoin.FieldOrder, "\", \""))
			}
			term += "}"
		} else if t.Representation.ListPairs != nil {
			term += " representation listpairs"
		}
	}
	return term
}

func (t *TypeBool) TypeDecl() string {
	return "bool"
}
func (t *TypeInt) TypeDecl() string {
	return "int"
}
func (t *TypeFloat) TypeDecl() string {
	return "float"
}
func (t *TypeMap) TypeDecl() string {
	tval := TypeTermDecl(t.ValueType)
	if t.ValueNullable {
		tval = "nullable " + tval
	}

	decl := fmt.Sprintf("{%s:%s}", t.KeyType, tval)

	if t.Representation != nil {
		if t.Representation.StringPairs != nil {
			decl += stringPairsRepresentationDecl(t.Representation.StringPairs)
		}
		if t.Representation.ListPairs != nil {
			decl += " representation listpairs"
		}
	}
	return decl
}

func stringPairsRepresentationDecl(repr *Representation_StringPairs) string {
	return fmt.Sprintf(" representation stringpairs {\n\tinnerDelim \"%s\"\n\tentryDelim \"%s\"\n}",
		repr.InnerDelim,
		repr.EntryDelim)
}

func (t *TypeList) TypeDecl() string {
	tval := TypeTermDecl(t.ValueType)
	if t.ValueNullable {
		tval = "nullable " + tval
	}
	return fmt.Sprintf("[%s]", tval)
}

func (t *TypeUnion) TypeDecl() string {
	term := "union {\n"
	if t.Representation != nil {
		if t.Representation.Envelope != nil {
			for k, v := range t.Representation.Envelope.DiscriminantTable {
				term += fmt.Sprintf("\t| %s \"%s\"\n", TypeTermDecl(v), k)
			}
			term += "} representation envelope {\n"
			term += fmt.Sprintf("\tdiscriminantKey \"%s\"\n\tcontentKey \"%s\"\n}",
				t.Representation.Envelope.DiscriminantKey,
				t.Representation.Envelope.ContentKey)
		} else if t.Representation.Keyed != nil {
			for k, v := range *(t.Representation.Keyed) {
				term += fmt.Sprintf("\t| %s \"%s\"\n", TypeTermDecl(v), k)
			}
			term += "} representation keyed"
		} else if t.Representation.Kinded != nil {
			for k, v := range *(t.Representation.Kinded) {
				term += fmt.Sprintf("\t| %s %s\n", TypeTermDecl(v), k)
			}

			term += "} representation kinded"
		} else if t.Representation.Inline != nil {
			for k, v := range t.Representation.Inline.DiscriminantTable {
				term += fmt.Sprintf("\t| %s %q\n", TypeTermDecl(v), k)
			}

			term += fmt.Sprintf("} representation inline {\n\tdiscriminantKey \"%s\"\n}",
				t.Representation.Inline.DiscriminantKey)
		} else if t.Representation.BytePrefix != nil {
			for k, v := range *(t.Representation.BytePrefix) {
				term += fmt.Sprintf("\t| %s %d\n", TypeTermDecl(k), v)
			}

			term += "} representation byteprefix"
		} else {
			panic(fmt.Sprintf("no representation type specified for union"))
		}
	}
	return term
}

func (t *TypeEnum) TypeDecl() string {
	term := "enum {\n"
	for m := range t.Members {
		term += fmt.Sprintf("\t| \"%s\"\n", m)
	}
	term += "}"
	return term
}

func (t *TypeLink) TypeDecl() string {
	if t.ExpectedType != nil {
		return fmt.Sprintf("&%s", TypeTermDecl(t.ExpectedType))
	}
	return "&Any"
}

func (t *TypeString) TypeDecl() string {
	return "string"
}

func (t *TypeBytes) TypeDecl() string {
	return "bytes"
}

func (t NamedType) TypeDecl() string {
	return string(t)
}

func (t *TypeCopy) TypeDecl() string {
	return fmt.Sprintf(" = %s", t.FromType)
}

func TypeTermDecl(t TypeTerm) string {
	switch t := t.(type) {
	case Type:
		return t.TypeDecl()
	default:
		panic("what is this")
	}
}

type NamedType string

func SimpleType(kind string) Type {
	switch kind {
	case "bool":
		return &TypeBool{kind}
	case "bytes":
		return &TypeBytes{Kind: kind}
	case "float":
		return &TypeFloat{kind}
	case "int":
		return &TypeInt{kind}
	case "link":
		return &TypeLink{nil, kind}
	case "string":
		return &TypeString{kind}
	}
	return nil
}

type StructRepresentation struct {
	Map         *StructRepresentation_Map        `json:"map,omitempty"`
	Tuple       *StructRepresentation_Tuple      `json:"tuple,omitempty"`
	StringPairs *Representation_StringPairs      `json:"stringpairs,omitempty"`
	StringJoin  *StructRepresentation_StringJoin `json:"stringjoin,omitempty"`
	ListPairs   *Representation_ListPairs        `json:"listpairs,omitempty"`
}

type StructRepresentation_Map struct {
	Fields map[string]SRMFieldDetails `json:"fields,omitempty"`
}

type StructRepresentation_Tuple struct {
	FieldOrder []string `json:"fieldOrder,omitempty"`
}

// shared by Map and String
type Representation_StringPairs struct {
	EntryDelim string `json:"entryDelim"`
	InnerDelim string `json:"innerDelim"`
}

type StructRepresentation_StringJoin struct {
	FieldOrder []string `json:"fieldOrder,omitempty"`
	Join       string   `json:"join,omitempty"`
}

// shared by Map and String
type Representation_ListPairs struct {
	FieldOrder []string `json:"fieldOrder,omitempty"`
}

type SRMFieldDetails struct {
	Implicit interface{} `json:"implicit,omitempty"`
	Rename   string      `json:"rename,omitempty"`
}

type StructField struct {
	Nullable bool     `json:"nullable,omitempty"`
	Optional bool     `json:"optional,omitempty"`
	Type     TypeTerm `json:"type"`
}

type TypeTerm interface{}

type TypeBool struct {
	Kind string `json:"kind"`
}

type TypeString struct {
	Kind string `json:"kind"`
}

type TypeBytes struct {
	Kind           string               `json:"kind"`
	Representation *BytesRepresentation `json:"representation,omitempty"`
}

type BytesRepresentation struct {
	Advanced *AdvancedDataLayoutName `json:"advanced,omitempty"`
}

type AdvancedDataLayoutName string

type TypeInt struct {
	Kind string `json:"kind"`
}

type TypeFloat struct {
	Kind string `json:"kind"`
}

type TypeLink struct {
	ExpectedType TypeTerm `json:"expectedType,omitempty"`
	Kind         string   `json:"kind"`
}

type TypeList struct {
	Kind           string              `json:"kind"`
	Representation *ListRepresentation `json:"representation,omitempty"`
	ValueNullable  bool                `json:"valueNullable,omitempty"`
	ValueType      TypeTerm            `json:"valueType,omitempty"`
}

type ListRepresentation struct {
	Advanced *AdvancedDataLayoutName `json:"advanced,omitempty"`
}

type TypeEnum struct {
	Kind    string               `json:"kind"`
	Members map[string]*struct{} `json:"members"`
}

type TypeUnion struct {
	Kind           string               `json:"kind"`
	Representation *UnionRepresentation `json:"representation,omitempty"`
}

type UnionRepresentation struct {
	Keyed      *UnionRepresentation_Keyed      `json:"keyed,omitempty"`
	Kinded     *UnionRepresentation_Kinded     `json:"kinded,omitempty"`
	Envelope   *UnionRepresentation_Envelope   `json:"envelope,omitempty"`
	Inline     *UnionRepresentation_Inline     `json:"inline,omitempty"`
	BytePrefix *UnionRepresentation_BytePrefix `json:"byteprefix,omitempty"`
}

type TypeName string
type UnionRepresentation_Keyed map[string]Type

type RepresentationKind string
type UnionRepresentation_Kinded map[RepresentationKind]Type

type UnionRepresentation_Envelope struct {
	ContentKey        string          `json:"contentKey"`
	DiscriminantKey   string          `json:"discriminantKey"`
	DiscriminantTable map[string]Type `json:"discriminantTable"`
}

type UnionRepresentation_BytePrefix map[NamedType]int

type UnionRepresentation_Inline struct {
	DiscriminantKey   string          `json:"discriminantKey"`
	DiscriminantTable map[string]Type `json:"discriminantTable"`
}

type MapRepresentation struct {
	Map         *MapRepresentation_Map      `json:"map,omitempty"`
	StringPairs *Representation_StringPairs `json:"stringpairs,omitempty"`
	ListPairs   *Representation_ListPairs   `json:"listpairs,omitempty"`
	Advanced    *AdvancedDataLayoutName     `json:"advanced,omitempty"`
}

type MapRepresentation_Map struct{}

type TypeCopy struct {
	FromType string `json:"fromType"`
	Kind string `json:"kind"`
}
