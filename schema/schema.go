package schema

import (
	"fmt"
)

type TypeStruct struct {
	Fields         map[string]*StructField
	Representation StructRepresentation
}

func (t *TypeStruct) TypeDecl() string {
	term := "struct {\n"

	fieldInfo := make(map[string]SRMFieldDetails)
	if srep, ok := t.Representation.(*StructRepresentation_Map); ok {
		fieldInfo = srep.Fields
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

		if inf, ok := fieldInfo[fname]; ok && (inf.Alias != "" || inf.Default != nil) {
			term += " ("
			if inf.Alias != "" {
				term += fmt.Sprintf("rename \"%s\"", inf.Alias)
			}
			if inf.Default != nil {
				if inf.Alias != "" {
					term += ", "
				}
				term += fmt.Sprintf("implicit %q", inf.Default)
			}
			term += ")"
		}
		term += "\n"
	}
	term += "}"
	if t.Representation != nil {
		switch t.Representation.(type) {
		case *StructRepresentation_Map:
			term += " representation map"
		case *StructRepresentation_Tuple:
			term += " representation tuple"
		}
	}
	return term
}

func (t *TypeBool) TypeDecl() string {
	return "Bool"
}
func (t *TypeInt) TypeDecl() string {
	return "Int"
}
func (t *TypeFloat) TypeDecl() string {
	return "Float"
}
func (t *TypeMap) TypeDecl() string {
	return fmt.Sprintf("{%s:%s}", t.KeyType, TypeTermDecl(t.ValueType))
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
	switch rep := t.Representation.(type) {
	case *UnionRepresentation_Envelope:
		panic("TODO")
	case UnionRepresentation_Keyed:
		for k, v := range rep {
			term += fmt.Sprintf("\t| %s %s\n", v, k)
		}

		term += "} representation keyed"
	case UnionRepresentation_Kinded:
		for k, v := range rep {
			term += fmt.Sprintf("\t| %s %s\n", v, k)
		}

		term += "} representation kinded"
	case *UnionRepresentation_Inline:
		for k, v := range rep.DiscriminantTable {
			term += fmt.Sprintf("\t| %s %q\n", v, k)
		}

		term += fmt.Sprintf("} representation inline \"%s\"", rep.DiscriminatorKey)
	default:
		panic(fmt.Sprintf("unrecognized type: %T", t.Representation))
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
	return fmt.Sprintf("&%s", TypeTermDecl(t.ValueType))
}

func (t *TypeString) TypeDecl() string {
	return "String"
}

func (t *TypeBytes) TypeDecl() string {
	return "Bytes"
}

func (t NamedType) TypeDecl() string {
	return string(t)
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

type StructRepresentation interface{}

type StructRepresentation_Map struct {
	Fields map[string]SRMFieldDetails
}

type StructRepresentation_Tuple struct {
	FieldOrder []string
}

type SRMFieldDetails struct {
	Alias   string
	Default interface{}
}

type StructField struct {
	Type     TypeTerm
	Optional bool
	Nullable bool
}

type TypeTerm interface{}

type TypeBool struct{}
type TypeString struct{}
type TypeBytes struct{}
type TypeInt struct{}
type TypeFloat struct{}

type TypeLink struct {
	ValueType TypeTerm
}

type TypeList struct {
	ValueType     TypeTerm
	ValueNullable bool
}

type TypeEnum struct {
	Members map[string]struct{}
}

type TypeUnion struct {
	Representation UnionRepresentation
}

type UnionRepresentation interface{}

type RepresentationKind string
type TypeName string

type UnionRepresentation_Kinded map[RepresentationKind]TypeName

type UnionRepresentation_Keyed map[string]TypeName

type UnionRepresentation_Envelope struct {
	DiscriminatorKey  string
	ContentKey        string
	DiscriminantTable map[string]TypeName
}

type UnionRepresentation_Inline struct {
	DiscriminatorKey  string
	DiscriminantTable map[string]TypeName
}

type SchemaMap map[string]Type

type Type interface {
	TypeDecl() string
}

type TypeMap struct {
	KeyType       string
	ValueType     TypeTerm
	ValueNullable bool
}
