package parser

import (
	"bufio"
	"fmt"

	. "github.com/whyrusleeping/ipld-schema/schema"
)

func tokens(l string) []string {
	var out []string
	curStart := -1

	var quoted bool
loop:
	for i := 0; i < len(l); i++ {
		if quoted && l[i] != '"' {
			continue
		}

		switch l[i] {
		case '"':
			if !quoted {
				quoted = true
				curStart = i + 1
			} else {
				out = append(out, l[curStart:i])
				curStart = -1
				quoted = false
			}
		case ' ', '\t':
			if curStart != -1 {
				out = append(out, l[curStart:i])
			}
			curStart = -1
		case '{', '[', '(':
			out = append(out, l[i:i+1])
		case '}', ']', ':', ')':
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
				return &TypeUnion{UnionRepresentation_Kinded{}}, nil
			case "inline":
				if len(toks) < 4 {
					return nil, fmt.Errorf("expected open bracket for inline union representation block")
				}
				urep, err := parseUnionInlineRepresentation(s)
				if err != nil {
					return nil, err
				}

				urep.DiscriminantTable = make(map[string]TypeName)
				for k, v := range unionVals {
					urep.DiscriminantTable[k] = TypeName(v)
				}

				return &TypeUnion{urep}, nil
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

func parseUnionInlineRepresentation(s *bufio.Scanner) (*UnionRepresentation_Inline, error) {
	var urep UnionRepresentation_Inline
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		switch toks[0] {
		case "discriminantKey":
			if urep.DiscriminatorKey != "" {
				return nil, fmt.Errorf("multiple discriminatorKeys in inline representation")
			}
			urep.DiscriminatorKey = toks[1]
		case "}":
			return &urep, nil
		default:
			return nil, fmt.Errorf("unrecognized token %q in inline union representation", toks[0])

		}
	}

	return nil, fmt.Errorf("reached end of file while parsing inline union representation")
}

func parseStruct(s *bufio.Scanner) (*TypeStruct, error) {
	st := &TypeStruct{
		Fields: make(map[string]*StructField),
	}
	freps := make(map[string]SRMFieldDetails)
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				if toks[1] == "representation" {
					repr, err := parseStructRepr(toks, s, freps)
					if err != nil {
						return nil, err
					}
					st.Representation = repr
				}
			}

			return st, nil
		}

		var frep *SRMFieldDetails
		if toks[len(toks)-1] == ")" {
			frepStart := 0
			for ; frepStart < len(toks); frepStart++ {
				if toks[frepStart] == "(" {
					break
				}
			}
			var err error
			frep, err = parseStructFieldRep(toks[frepStart+1 : len(toks)-1])
			if err != nil {
				return nil, err
			}
			toks = toks[:frepStart]
		}

		fname, strf, err := parseStructField(toks)
		if err != nil {
			return nil, err
		}

		if frep != nil {
			freps[fname] = *frep
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

func parseStructFieldRep(toks []string) (*SRMFieldDetails, error) {
	var fd SRMFieldDetails
	applyInfo := func(k, v string) error {
		switch k {
		case "implicit":
			if fd.Default != nil {
				return fmt.Errorf("duplicate implicit in struct field representation")
			}
			fd.Default = v
		case "rename":
			if fd.Alias != "" {
				return fmt.Errorf("duplicate alias in struct field representation")
			}
			fd.Alias = v
		default:
			return fmt.Errorf("unrecognized struct field representation key: %s", k)
		}
		return nil
	}

	switch len(toks) {
	case 5:
		if err := applyInfo(toks[3], toks[4]); err != nil {
			return nil, err
		}
		fallthrough
	case 2:
		if err := applyInfo(toks[0], toks[1]); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("incorrectly formatted struct field representation: %q", toks)
	}
	return &fd, nil
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
			return NamedType(toks[0]), nil
		}
		fmt.Println("bad bad bad: ", toks[0])
		fmt.Println("Full line: ")
		fmt.Println(toks)
		panic("cant deal")
	}
}

func parseStructRepr(line []string, s *bufio.Scanner, freps map[string]SRMFieldDetails) (StructRepresentation, error) {
	if len(line) < 3 {
		return nil, fmt.Errorf("no representation kind given")
	}

	kind := line[2]
	switch kind {
	case "tuple":
		if len(freps) > 0 {
			return nil, fmt.Errorf("tuple struct representation cannot have field details")
		}
		if len(line) == 4 {
			return nil, fmt.Errorf("cant yet handle detailed tuple representation")
		}

		return &StructRepresentation_Tuple{}, nil
	case "map":
		if len(line) > 3 {
			return nil, fmt.Errorf("unexpected tokens after 'representation map'")
		}

		return &StructRepresentation_Map{Fields: freps}, nil
	default:
		return nil, fmt.Errorf("unrecognized struct representation: %s", kind)
	}
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
