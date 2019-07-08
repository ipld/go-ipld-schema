package schema

import (
	"fmt"
	"io"
	"sort"
)

func ExportIpldSchema(sm SchemaMap, w io.Writer) error {
	var types []string
	for tname := range sm {
		types = append(types, tname)
	}

	sort.Strings(types)

	for _, tname := range types {
		t := sm[tname]
		fmt.Fprintf(w, "type %s %s\n\n", tname, t.TypeDecl())
	}

	return nil
}
