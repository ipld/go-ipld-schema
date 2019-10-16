package gengo

import (
	"fmt"
	"io"
	"sort"
	"strings"

	. "github.com/ipld/go-ipld-schema/schema"
)

// in reality, the 'right' way to do this is to probably use the golang ast packages
func GolangCodeGen(schema *Schema, w io.Writer) error {
	var types []string
	for tname := range schema.TypesMap {
		types = append(types, tname)
	}

	sort.Strings(types)
	fmt.Fprintf(w, "package main\n\n")

	for _, tname := range types {
		t := schema.TypesMap[tname]
		tname := strings.Title(tname)
		switch t := t.(type) {
		case *TypeStruct:
			fmt.Fprintf(w, "type %s struct {\n", tname)
			for fname, f := range t.Fields {
				t := typeToGoType(f.Type.(Type)) // TODO: should TypeTerm just be type? it seems like it wants that
				if f.Nullable {
					t = "*" + t
				}
				fname := strings.Title(fname)
				fmt.Fprintf(w, "\t%s %s\n", fname, t)
			}
			fmt.Fprintf(w, "}\n\n")
		case *TypeEnum:
			enumTag := "_Enum" + tname
			fmt.Fprintf(w, "type %s interface {\n\t%s()\n}\n", tname, enumTag)
			for mem := range t.Members {
				enumelem := tname + mem
				fmt.Fprintf(w, "type %s struct{}\n", enumelem)
				fmt.Fprintf(w, "func (_ %s) %s() {}\n", enumelem, enumTag)
				fmt.Fprintf(w, "var _ %s = (*%s)(nil)\n", tname, enumelem)
			}
		case *TypeUnion:
			fmt.Fprintf(w, "type %s interface {}\n", tname)
			/*
				switch rep := t.Representation.(type) {

				}
			*/
		default:
			fmt.Fprintf(w, "type %s %s\n\n", tname, typeToGoType(t))
		}
	}
	return nil
}

func typeToGoType(t Type) string {
	switch t := t.(type) {
	case *TypeBool:
		return "bool"
	case *TypeString:
		return "string"
	case *TypeBytes:
		return "[]byte"
	case *TypeInt:
		return "int"
	case *TypeFloat:
		return "float64"

	case *TypeLink:
		et := "Any"
		if t.ExpectedType != nil {
			et = fmt.Sprintf("%s", t.ExpectedType)
		}
		return fmt.Sprintf("cid.Cid /* IPLD: %s */", et)

	case *TypeList:
		subtype := typeToGoType(t.ValueType.(Type)) // TypeTerm really wants to be a Type
		if t.ValueNullable {
			subtype = "*" + subtype
		}
		return fmt.Sprintf("[]%s", subtype)

	case *TypeEnum:
		panic("no")
	case *TypeUnion:
		panic("no")
	case *TypeMap:
		val := typeToGoType(t.ValueType.(Type))
		if t.ValueNullable {
			val = "*" + val
		}
		return fmt.Sprintf("map[%s]%s", typeToGoType(NamedType(t.KeyType)), val)
	case NamedType:
		return string(t)
	default:
		fmt.Printf("BAD TYPE: %T\n", t)
		panic("unrecognized type")
	}
}
