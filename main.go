package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

type TypeStruct struct {
	Fields         map[string]*StructField
	Representation StructRepresentation
}

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
}

type TypeMap struct {
	KeyType       string
	ValueType     TypeTerm
	ValueNullable bool
}

func tokens(l string) []string {
	var out []string
	curStart := -1

loop:
	for i := 0; i < len(l); i++ {
		switch l[i] {
		case ' ', '\t':
			if curStart != -1 {
				out = append(out, l[curStart:i])
			}
			curStart = -1
		case '{', '[':
			out = append(out, l[i:i+1])
		case '}', ']', ':':
			if curStart != -1 {
				out = append(out, l[curStart:i])
			}
			out = append(out, l[i:i+1])
			curStart = -1
		case '#':
			break loop
		default:
			if curStart == -1 {
				curStart = i
			}
		}
	}
	if curStart != -1 {
		out = append(out, l[curStart:])
	}

	return out
}

/*
func tokens(l string) []string {
	l = strings.TrimSpace(l)
	if strings.HasPrefix(l, "#") {
		return nil
	}

	var out []string
	for _, s := range strings.Split(l, " ") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
*/

func parseType(tline []string, s *bufio.Scanner) (string, Type, error) {
	if len(tline) < 3 {
		return "", nil, fmt.Errorf("expected at least three tokens on type definition line")
	}

	// thinking we should just call 'parseTypeTerm' here...

	tname := tline[1]
	ttype := tline[2]

	var t Type
	var err error
	switch ttype {
	case "struct":
		if len(tline) < 4 || tline[3] != "{" {
			return "", nil, fmt.Errorf("struct declaration must contain an open brace")
		}

		if len(tline) > 4 {
			// parse fucky struct declaration
			if tline[len(tline)-1] != "}" {
				return "", nil, fmt.Errorf("oneline struct declaration must terminate on same line")
			}

			inner := tline[4 : len(tline)-1]
			if len(inner) == 0 {
				return tname, &TypeStruct{}, nil
			}

			fname, strf, err := parseStructField(inner)
			if err != nil {
				return "", nil, err
			}

			return tname, &TypeStruct{
				Fields: map[string]*StructField{
					fname: strf,
				},
			}, nil
		}

		t, err = parseStruct(s)
	case "union":
		if len(tline) != 4 || tline[3] != "{" {
			return "", nil, fmt.Errorf("union declaration must end in an open brace")
		}
		t, err = parseUnion(s)
	case "enum":
		if len(tline) != 4 || tline[3] != "{" {
			return "", nil, fmt.Errorf("enum declaration must end in an open brace")
		}
		t, err = parseEnum(s)
	default:
		t, err = parseTypeTerm(tline[2:])
	}

	return tname, t, err
}

func parseEnum(s *bufio.Scanner) (*TypeEnum, error) {
	vals := make(map[string]struct{})
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		switch toks[0] {
		case "|":
			vals[toks[1]] = struct{}{}
		case "}":
			return &TypeEnum{
				Members: vals,
			}, nil
		default:
			return nil, fmt.Errorf("unexpected token: %s", toks[0])
		}
	}
	return nil, fmt.Errorf("unterminated enum")
}

func parseUnion(s *bufio.Scanner) (*TypeUnion, error) {
	unionVals := make(map[string]string)
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		switch toks[0] {
		case "|":
			if len(toks) != 3 {
				return nil, fmt.Errorf("must have three tokens in union entry")
			}
			typeName := toks[1]
			key := toks[2]
			unionVals[key] = typeName
		case "}":
			if len(toks) < 3 {
				return nil, fmt.Errorf("union closing line must contain at least three tokens")
			}

			if toks[1] != "representation" {
				return nil, fmt.Errorf("must specify union representation")
			}

			switch toks[2] {
			case "kinded":
				return &TypeUnion{&UnionRepresentation_Kinded{}}, nil
			case "inline":
				rep := &UnionRepresentation_Inline{
					DiscriminatorKey:  toks[3],
					DiscriminantTable: make(map[string]TypeName),
				}

				for k, v := range unionVals {
					rep.DiscriminantTable[k] = TypeName(v)
				}

				return &TypeUnion{rep}, nil
			case "keyed":
				rep := make(UnionRepresentation_Keyed)
				for k, v := range unionVals {
					rep[k] = TypeName(v)
				}
				return &TypeUnion{rep}, nil
			case "envelope":
				return nil, fmt.Errorf("Not Yet Implemented: enveloped unions")
			}

		}
	}

	return nil, fmt.Errorf("unterminated union declaration")
}

func parseStruct(s *bufio.Scanner) (*TypeStruct, error) {
	st := &TypeStruct{
		Fields: make(map[string]*StructField),
	}
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				if toks[1] == "representation" {
					repr, err := parseStructRepr(toks, s)
					if err != nil {
						return nil, err
					}
					st.Representation = repr
				}
			}

			return st, nil
		}

		fname, strf, err := parseStructField(toks)
		if err != nil {
			return nil, err
		}

		st.Fields[fname] = strf
	}

	return st, nil
}

func parseStructField(toks []string) (string, *StructField, error) {
	var optional, nullable bool
	var i int = 1

loop:
	for ; i < len(toks)-1; i++ {
		switch toks[i] {
		case "optional":
			if optional {
				return "", nil, fmt.Errorf("multiple optional keywords")
			}
			optional = true
		case "nullable":
			if nullable {
				return "", nil, fmt.Errorf("multiple nullable keywords")
			}
			nullable = true
		default:
			break loop
		}
	}

	trepr, err := parseTypeTerm(toks[i:])
	if err != nil {
		return "", nil, err
	}

	fname := toks[0]

	return fname, &StructField{
		Type:     trepr,
		Nullable: nullable,
		Optional: optional,
	}, nil
}

func parseTypeTerm(toks []string) (Type, error) {
	if len(toks) == 0 {
		return nil, fmt.Errorf("no tokens for type term")
	}

	switch toks[0] {
	case "[":
		toks = toks[1:]

		last := toks[len(toks)-1]
		if last != "]" {
			return nil, fmt.Errorf("TypeTerm must end with matching ']'")
		}
		toks = toks[:len(toks)-1]

		var nullable bool
		if toks[0] == "nullable" {
			nullable = true
			toks = toks[1:]
		}

		subtype, err := parseTypeTerm(toks)
		if err != nil {
			return nil, err
		}

		return &TypeList{
			ValueType:     subtype,
			ValueNullable: nullable,
		}, nil
	case "{":
		if len(toks) < 5 {
			// not a great error message, should more clearly tell user
			// what is actually missing
			return nil, fmt.Errorf("map TypeTerms must be at least 5 tokens")
		}

		if toks[len(toks)-1] != "}" {
			return nil, fmt.Errorf("map TypeTerm must end with matching '}'")
		}

		typeName := toks[1]
		if toks[2] != ":" {
			return nil, fmt.Errorf("expected ':' between map key type and value type")
		}

		valTermStart := 3
		var nullable bool
		if toks[3] == "nullable" {
			nullable = true
			valTermStart++
		}

		valt, err := parseTypeTerm(toks[valTermStart : len(toks)-1])
		if err != nil {
			return nil, err
		}

		return &TypeMap{
			KeyType:       typeName,
			ValueType:     valt,
			ValueNullable: nullable,
		}, nil
	case "&":
		toks = toks[1:]

		linktype, err := parseTypeTerm(toks)
		if err != nil {
			return nil, err
		}

		return &TypeLink{
			ValueType: linktype,
		}, nil
	default:
		if len(toks) == 1 {
			return toks[0], nil
		}
		fmt.Println("bad bad bad: ", toks[0])
		fmt.Println("Full line: ")
		fmt.Println(toks)
		panic("cant deal")
	}
}

func parseStructRepr(line []string, s *bufio.Scanner) (StructRepresentation, error) {
	if len(line) < 3 {
		return nil, fmt.Errorf("no representation kind given")
	}

	kind := line[2]
	switch kind {
	case "tuple":
		if len(line) == 4 {
			return nil, fmt.Errorf("cant yet handle detailed tuple representation")
		}

		return &StructRepresentation_Tuple{}, nil
	case "map":
		if len(line) != 4 || line[3] != "{" {
			return nil, fmt.Errorf("expected an open brace")
		}

		return parseStructRepMap(s)
	default:
		return nil, fmt.Errorf("unrecognized struct representation: %s", kind)
	}
}

func parseStructRepMap(s *bufio.Scanner) (*StructRepresentation_Map, error) {
	srm := &StructRepresentation_Map{
		Fields: make(map[string]SRMFieldDetails),
	}
	for s.Scan() {
		toks := tokens(s.Text())
		switch toks[0] {
		case "}":
			return srm, nil
		case "field":
			var fdet SRMFieldDetails
			fname := toks[1]
			switch toks[2] {
			case "alias":
				fdet.Alias = toks[2]
			case "default":
				fdet.Default = toks[2]
			default:
				return nil, fmt.Errorf("unrecognized field representation option: %s", toks[2])
			}
			srm.Fields[fname] = fdet
		default:
			return nil, fmt.Errorf("unrecognized token in struct map representation: %s", toks[0])
		}
	}
	return nil, fmt.Errorf("unterminated struct map representation")
}

func ParseSchema(s *bufio.Scanner) (SchemaMap, error) {
	sm := make(SchemaMap)
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		switch toks[0] {
		case "type":
			name, t, err := parseType(toks, s)
			if err != nil {
				fmt.Println("failed to parse line: ")
				fmt.Println(s.Text())
				fmt.Printf("%q\n", toks)
				return nil, err
			}
			sm[name] = t
		default:
			return nil, fmt.Errorf("unexpected token: %q", toks[0])
		}
	}
	return sm, nil
}

func main() {
	//fmt.Printf("%q\n", tokens("bar [Int]"))
	//return
	/*
			testval := `
		type FooBar struct {
		  cat CatType
		  bar [Int]
		  baz &Whatever
		} representation tuple
		`
	*/

	fi, err := os.Open("schema-schema.ipldsch")
	if err != nil {
		panic(err)
	}
	defer fi.Close()

	s := bufio.NewScanner(fi)
	schema, err := ParseSchema(s)
	if err != nil {
		panic(err)
	}

	fmt.Println(schema)
	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	if err := GolangCodeGen(schema, os.Stdout); err != nil {
		panic(err)
	}
}

/// SEPARATE FILE

// in reality, the 'right' way to do this is to probably use the golang ast packages
func GolangCodeGen(sm SchemaMap, w io.Writer) error {
	var types []string
	for tname := range sm {
		types = append(types, tname)
	}

	sort.Strings(types)

	for _, tname := range types {
		t := sm[tname]
		switch t := t.(type) {
		case *TypeStruct:
			fmt.Fprintf(w, "type %s struct {\n", tname)
			for fname, f := range t.Fields {
				t := typeToGoType(f.Type)
				if f.Nullable {
					t = "*" + t
				}
				fmt.Fprintf(w, "\t%s %s\n", fname, t)
			}
			fmt.Fprintf(w, "}\n\n")
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
		return fmt.Sprintf("cid.Cid /* IPLD: %s */", typeToGoType(t))

	case *TypeList:
		subtype := typeToGoType(t.ValueType)
		if t.ValueNullable {
			subtype = "*" + subtype
		}
		return fmt.Sprintf("[]%s", subtype)

	case *TypeEnum:
		panic("no")
	case *TypeUnion:
		panic("no")
	case *TypeMap:
		val := typeToGoType(t.ValueType)
		if t.ValueNullable {
			val = "*" + val
		}
		return fmt.Sprintf("map[%s]%s", typeToGoType(t.KeyType), val)
	case string:
		return t
	default:
		fmt.Printf("BAD TYPE: %T\n", t)
		panic("unrecognized type")
	}
}
