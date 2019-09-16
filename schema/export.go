package schema

import (
	"fmt"
	"io"
	"sort"
)

func ExportIpldSchema(sch *Schema, w io.Writer) error {
	var types []string
	for tname := range sch.SchemaMap {
		types = append(types, tname)
	}

	sort.Strings(types)

	for _, tname := range types {
		t := sch.SchemaMap[tname]
		fmt.Fprintf(w, "type %s %s\n\n", tname, t.TypeDecl())
	}

	return nil
}
