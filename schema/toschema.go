package schema

import (
	"fmt"
	"io"
	"strings"
)

func PrintSchema(sch *Schema, w io.Writer) error {
	if sch.AdvancedList != nil {
		for _, a := range sch.AdvancedList.AsList() {
			fmt.Fprintf(w, "advanced %s\n\n", a.GetName())
		}
	}

	for _, t := range sch.TypesList.AsList() {
		fmt.Fprintf(w, "type %s %s\n\n", t.GetName(), t.TypeDecl())
	}

	return nil
}

func TypeTermDecl(t TypeTerm) string {
	switch t := t.(type) {
	case Type:
		return t.TypeDecl()
	default:
		panic("Got something unexpected without a TypeDecl()")
	}
}

func (t *TypeStruct) TypeDecl() string {
	term := "struct {"

	if t.Fields.Count() > 0 {
		term += "\n"

		for _, f := range t.Fields.AsList() {
			term += fmt.Sprintf("\t%s", f.GetName())
			if f.Optional {
				term += " optional"
			}
			if f.Nullable {
				term += " nullable"
			}
			term += " "
			term += TypeTermDecl(f.Type)

			if t.Representation != nil && t.Representation.Map != nil {
				if inf, ok := t.Representation.Map.GetFieldDetails(f.GetName()); ok && (inf.Rename != "" || inf.Implicit != nil) {
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
			}
			term += "\n"
		}
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

func (t *TypeMap) TypeDecl() string {
	tval := TypeTermDecl(t.ValueType)
	if t.ValueNullable {
		tval = "nullable " + tval
	}

	decl := fmt.Sprintf("{%s:%s}", t.KeyType, tval)

	if t.Representation != nil {
		if t.Representation.StringPairs != nil {
			decl += stringPairsRepresentationDecl(t.Representation.StringPairs)
		} else if t.Representation.ListPairs != nil {
			decl += " representation listpairs"
		} else if t.Representation.Advanced != nil {
			decl += fmt.Sprintf(" representation advanced %s", *t.Representation.Advanced)
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
	term := fmt.Sprintf("[%s]", tval)

	if t.Representation != nil && t.Representation.Advanced != nil {
		term += fmt.Sprintf(" representation advanced %s", *t.Representation.Advanced)
	}

	return term
}

func (t *TypeEnum) TypeDecl() string {
	term := "enum {\n"
	for _, m := range t.Members.AsList() {
		term += fmt.Sprintf("\t| %s", m)
		if t.Representation != nil {
			if t.Representation.String != nil {
				if val, ok := t.Representation.String.GetMapping(m); ok {
					term += fmt.Sprintf(" (\"%s\")", val)
				}
			} else if t.Representation.Int != nil {
				if val, ok := t.Representation.Int.GetMapping(m); ok {
					term += fmt.Sprintf(" (\"%d\")", val)
				}
			}
		}
		term += "\n"
	}
	term += "}"
	if t.Representation != nil && t.Representation.Int != nil {
		term += " representation int"
	}
	return term
}

func (t *TypeUnion) TypeDecl() string {
	term := "union {\n"
	if t.Representation != nil {
		if t.Representation.Envelope != nil {
			for _, k := range t.Representation.Envelope.DiscriminantTable.DiscriminantList() {
				v, _ := t.Representation.Envelope.DiscriminantTable.GetMapping(k)
				term += fmt.Sprintf("\t| %s \"%s\"\n", TypeTermDecl(v), k)
			}
			term += "} representation envelope {\n"
			term += fmt.Sprintf("\tdiscriminantKey \"%s\"\n\tcontentKey \"%s\"\n}",
				t.Representation.Envelope.DiscriminantKey,
				t.Representation.Envelope.ContentKey)
		} else if t.Representation.Keyed != nil {
			for _, k := range t.Representation.Keyed.KeyList() {
				v, _ := t.Representation.Keyed.GetMapping(k)
				term += fmt.Sprintf("\t| %s \"%s\"\n", TypeTermDecl(v), k)
			}
			term += "} representation keyed"
		} else if t.Representation.Kinded != nil {
			for _, k := range t.Representation.Kinded.KindList() {
				v, _ := t.Representation.Kinded.GetMapping(k)
				term += fmt.Sprintf("\t| %s %s\n", TypeTermDecl(v), k)
			}

			term += "} representation kinded"
		} else if t.Representation.Inline != nil {
			for _, k := range t.Representation.Inline.DiscriminantTable.DiscriminantList() {
				v, _ := t.Representation.Inline.DiscriminantTable.GetMapping(k)
				term += fmt.Sprintf("\t| %s %q\n", TypeTermDecl(v), k)
			}

			term += fmt.Sprintf("} representation inline {\n\tdiscriminantKey \"%s\"\n}",
				t.Representation.Inline.DiscriminantKey)
		} else if t.Representation.BytePrefix != nil {
			for _, k := range t.Representation.BytePrefix.TypeList() {
				v, _ := t.Representation.BytePrefix.GetMapping(k)
				term += fmt.Sprintf("\t| %s %d\n", TypeTermDecl(k), v)
			}

			term += "} representation byteprefix"
		} else {
			panic("no representation type specified for union")
		}
	}
	return term
}

func (t NamedType) TypeDecl() string {
	return string(t.name)
}

func (t TypeBool) TypeDecl() string {
	return "bool"
}

func (t TypeString) TypeDecl() string {
	return "string"
}

func (t TypeNull) TypeDecl() string {
	return "null"
}

func (t TypeBytes) TypeDecl() string {
	term := "bytes"

	if t.Representation != nil && t.Representation.Advanced != nil {
		term += fmt.Sprintf(" representation advanced %s", *t.Representation.Advanced)
	}

	return term
}

func (t TypeInt) TypeDecl() string {
	return "int"
}

func (t TypeFloat) TypeDecl() string {
	return "float"
}

func (t TypeLink) TypeDecl() string {
	if t.ExpectedType != nil {
		return fmt.Sprintf("&%s", TypeTermDecl(t.ExpectedType))
	}
	return "&Any"
}

func (t TypeCopy) TypeDecl() string {
	return fmt.Sprintf("= %s", t.FromType)
}
