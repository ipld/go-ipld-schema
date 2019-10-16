package parser

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	. "github.com/ipld/go-ipld-schema/schema"
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
				return tname, &TypeStruct{
					Kind:           "struct",
					Fields:         map[string]*StructField{},
					Representation: &StructRepresentation{Map: &StructRepresentation_Map{}},
				}, nil
			}

			fname, strf, err := parseStructField(inner)
			if err != nil {
				return "", nil, err
			}

			return tname, &TypeStruct{
				Kind: "struct",
				Fields: map[string]*StructField{
					fname: strf,
				},
				Representation: &StructRepresentation{Map: &StructRepresentation_Map{}},
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
	case "bool", "bytes", "float", "int", "link", "string":
		if ttype == "bytes" && len(tline) >= 4 && tline[3] == "representation" {
			if len(tline) != 6 || tline[4] != "advanced" {
				return "", nil, fmt.Errorf("%s declaration has malformed 'advanced' representation", ttype)
			}

			adlName := AdvancedDataLayoutName(tline[5])
			return tname, &TypeBytes{
				Kind:           "bytes",
				Representation: &BytesRepresentation{Advanced: &adlName},
			}, nil
		}

		if len(tline) == 3 {
			return tname, SimpleType(ttype), nil
		}

		return "", nil, fmt.Errorf("%s declaration cannot be followed by additional tokens", ttype)
	case "{":
		t, err = parseMapType(tline, s)
		if err != nil {
			return "", nil, err
		}
	case "[":
		t, err = parseListType(tline, s)
		if err != nil {
			return "", nil, err
		}
	case "=":
		if len(tline) != 4 {
			return "", nil, fmt.Errorf("%s copy type declaration requires a fromType type name", tname)
		}
		return tname, &TypeCopy{Kind: "copy", FromType: tline[3]}, nil
	default:
		t, err = parseTypeTerm(tline[2:])
	}

	return tname, t, err
}

func parseEnum(s *bufio.Scanner) (*TypeEnum, error) {
	vals := make(map[EnumValue]*struct{})
	reprVals := make(map[EnumValue]string)
	for s.Scan() {
		toks := tokens(s.Text())
		ntoks := len(toks)

		if ntoks == 0 { // blank line
			continue
		}

		if toks[0] == "|" {
			if ntoks == 1 {
				return nil, fmt.Errorf("expected EnumValue after '|'")
			}

			if ntoks != 2 && ntoks != 5 {
				return nil, fmt.Errorf("unexpected tokens after EnumValue %v", toks)
			}

			ev := EnumValue(toks[1])

			vals[ev] = nil

			if ntoks == 5 {
				if toks[2] != "(" && toks[4] != ")" {
					return nil, fmt.Errorf("unexpected tokens after EnumValue, expected representation value")
				}

				reprVals[ev] = toks[3]
			}

			continue
		}

		if toks[0] == "}" {
			er := EnumRepresentation{}

			if ntoks == 3 && toks[1] == "representation" && (toks[2] == "int" || toks[2] == "string") {
				if toks[2] == "int" {
					eri := make(EnumRepresentation_Int)
					for k, v := range reprVals {
						i, err := strconv.ParseInt(string(v), 10, 64)
						if err != nil {
							return nil, fmt.Errorf("'int' union representation may only use int values (%v)", v)
						}
						eri[k] = int(i)
					}
					er.Int = &eri
				}
			} else if ntoks != 1 {
				return nil, fmt.Errorf("unexpected tokens after end of enum")
			}

			if er.Int == nil {
				ers := make(EnumRepresentation_String)
				for k, v := range reprVals {
					ers[k] = v
				}
				er.String = &ers
			}

			return &TypeEnum{
				Kind:           "enum",
				Members:        vals,
				Representation: &er,
			}, nil
		}

		return nil, fmt.Errorf("unexpected token: %s", toks[0])
	}

	return nil, fmt.Errorf("unterminated enum")
}

func parseUnion(s *bufio.Scanner) (*TypeUnion, error) {
	unionVals := make(map[string]Type)
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
			key := toks[2]
			if toks[1][0] == '&' {
				unionVals[key] = tokenToLink(toks[1])
			} else {
				// TODO: validate characters
				unionVals[key] = NamedType(toks[1])
			}
		case "}":
			if len(toks) < 3 {
				return nil, fmt.Errorf("union closing line must contain at least three tokens")
			}

			if toks[1] != "representation" {
				return nil, fmt.Errorf("must specify union representation")
			}

			switch toks[2] {
			case "kinded":
				rep := make(UnionRepresentation_Kinded)
				for k, v := range unionVals {
					rep[RepresentationKind(k)] = v
				}
				repr := &UnionRepresentation{Kinded: &rep}
				return &TypeUnion{Kind: "union", Representation: repr}, nil
			case "inline":
				if len(toks) < 4 {
					return nil, fmt.Errorf("expected open bracket for inline union representation block")
				}
				urep, err := parseUnionInlineRepresentation(s)
				if err != nil {
					return nil, err
				}

				urep.DiscriminantTable = make(map[string]Type)
				for k, v := range unionVals {
					urep.DiscriminantTable[k] = v
				}

				return &TypeUnion{Kind: "union", Representation: &UnionRepresentation{Inline: urep}}, nil
			case "keyed":
				rep := make(UnionRepresentation_Keyed)
				for k, v := range unionVals {
					rep[k] = v
				}
				return &TypeUnion{Kind: "union", Representation: &UnionRepresentation{Keyed: &rep}}, nil
			case "envelope":
				if len(toks) < 4 {
					return nil, fmt.Errorf("expected open bracket for envelope union representation block")
				}
				urep, err := parseUnionEnvelopeRepresentation(s)
				if err != nil {
					return nil, err
				}

				urep.DiscriminantTable = make(map[string]Type)
				for k, v := range unionVals {
					urep.DiscriminantTable[k] = v
				}

				return &TypeUnion{Kind: "union", Representation: &UnionRepresentation{Envelope: urep}}, nil
			case "byteprefix":
				rep := make(UnionRepresentation_BytePrefix)
				for k, v := range unionVals {
					nt, ok := v.(NamedType)
					if !ok {
						return nil, fmt.Errorf("'byteprefix' union representation may only contain named types (%v)", v)
					}
					i, err := strconv.ParseInt(string(k), 10, 64)
					if err != nil {
						return nil, fmt.Errorf("'byteprefix' union representation may only use int discriminators (%v)", v)
					}
					rep[nt] = int(i)
				}
				return &TypeUnion{Kind: "union", Representation: &UnionRepresentation{BytePrefix: &rep}}, nil
			default:
				return nil, fmt.Errorf("unknown union representation '%s'", toks[2])
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
			if urep.DiscriminantKey != "" {
				return nil, fmt.Errorf("multiple 'discriminantKey's in inline representation")
			}
			urep.DiscriminantKey = toks[1]
		case "}":
			return &urep, nil
		default:
			return nil, fmt.Errorf("unrecognized token %q in inline representation", toks[0])
		}
	}

	return nil, fmt.Errorf("reached end of file while parsing inline union representation")
}

func parseUnionEnvelopeRepresentation(s *bufio.Scanner) (*UnionRepresentation_Envelope, error) {
	var urep UnionRepresentation_Envelope
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				return nil, fmt.Errorf("extraneous tokens found at end of envelope representation block [%v]", toks[1:])
			}
			if urep.DiscriminantKey == "" {
				return nil, fmt.Errorf("no 'discriminantKey' in envelope representation")
			}
			if urep.ContentKey == "" {
				return nil, fmt.Errorf("no 'contentKey' in envelope representation")
			}
			return &urep, nil
		}

		if len(toks) != 2 {
			return nil, fmt.Errorf("invalid tokens found in envelope representation block [%v]", toks)
		}

		switch toks[0] {
		case "discriminantKey":
			if urep.DiscriminantKey != "" {
				return nil, fmt.Errorf("multiple 'discriminantKey's in envelope representation")
			}
			urep.DiscriminantKey = toks[1]
		case "contentKey":
			if urep.ContentKey != "" {
				return nil, fmt.Errorf("multiple 'contentKey's in envelope representation")
			}
			urep.ContentKey = toks[1]
		default:
			return nil, fmt.Errorf("unrecognized token %q in envelope representation", toks[0])
		}
	}

	return nil, fmt.Errorf("reached end of file while parsing inline representation")
}

func parseStruct(s *bufio.Scanner) (*TypeStruct, error) {
	st := &TypeStruct{
		Kind:   "struct",
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
				} else {
					return nil, fmt.Errorf("extraneous tokens found at end of struct block [%v]", toks[1])
				}
			}
			if st.Representation == nil {
				// default representation
				st.Representation = &StructRepresentation{Map: &StructRepresentation_Map{Fields: freps}}
			}

			return st, nil
		}

		var srmFieldTokens []string
		typeTermFieldTokens := toks

		var frep *SRMFieldDetails
		if toks[len(toks)-1] == ")" {
			frepStart := 0
			for ; frepStart < len(toks); frepStart++ {
				if toks[frepStart] == "(" {
					break
				}
			}
			srmFieldTokens = toks[frepStart+1 : len(toks)-1]
			typeTermFieldTokens = toks[:frepStart]
		}

		fname, strf, err := parseStructField(typeTermFieldTokens)
		if err != nil {
			return nil, err
		}

		if len(srmFieldTokens) > 0 {
			var err error
			frep, err = parseStructFieldRep(strf.Type, srmFieldTokens)
			if err != nil {
				return nil, err
			}
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

func parseStructFieldRep(typeTerm TypeTerm, toks []string) (*SRMFieldDetails, error) {
	var fd SRMFieldDetails
	applyInfo := func(k, v string) error {
		switch k {
		case "implicit":
			if fd.Implicit != nil {
				return fmt.Errorf("duplicate implicit in struct field representation")
			}
			cv, err := coerceImplicitValueType(typeTerm, v)
			if err != nil {
				return err
			}
			fd.Implicit = cv
		case "rename":
			if fd.Rename != "" {
				return fmt.Errorf("duplicate rename in struct field representation")
			}
			fd.Rename = v
		default:
			return fmt.Errorf("unrecognized struct field representation key: %s", k)
		}
		return nil
	}

	switch len(toks) {
	case 4:
		if err := applyInfo(toks[2], toks[3]); err != nil {
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

func coerceImplicitValueType(typeTerm TypeTerm, value string) (interface{}, error) {
	nt, ok := typeTerm.(NamedType)
	if !ok {
		return nil, fmt.Errorf("Cannot accept implicit values for complex types [%v]", typeTerm)
	}

	switch nt {
	case "Bool":
		if value == "true" {
			return true, nil
		} else if value == "false" {
			return false, nil
		} else {
			return nil, fmt.Errorf("Could not convert implicit value [%s] in struct field representation to Bool", value)
		}
	case "Int":
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Could not convert implicit value [%s] in struct field representation to Int", value)
		}
		return i, nil
	case "String":
		return value, nil
	}

	return nil, fmt.Errorf("Could not convert implicit value struct field representation to correct type [%v]", typeTerm)
}

func parseTypeTerm(toks []string) (Type, error) {
	if len(toks) == 0 {
		return nil, fmt.Errorf("no tokens for type term")
	}

	if toks[0][0] == '&' {
		if len(toks) != 1 {
			return nil, fmt.Errorf("extraneous tokens after &Link declaration")
		}

		if len(toks[0]) == 1 {
			return nil, fmt.Errorf("invalid link type, '&' must be directly followed by an expected type string")
		}

		return tokenToLink(toks[0]), nil
	}

	// handle anonymous [] and {} types here
	switch toks[0] {
	case "[":
		return parseListTypeTerm(toks)
	case "{":
		return parseMapTypeTerm(toks)
	default:
		if len(toks) == 1 {
			return NamedType(toks[0]), nil
		}
		// TODO: better error
		fmt.Println("failed to parse token: ", toks[0])
		fmt.Println("full line: ")
		fmt.Println(toks)
		panic("Can't deal")
	}
}

func parseMapType(toks []string, s *bufio.Scanner) (*TypeMap, error) {
	end := len(toks)
	if end >= 8 && toks[7] == "representation" {
		end = 7
	} else if end >= 9 && toks[8] == "representation" { // could have a "nullable"
		end = 8
	}

	mt, err := parseMapTypeTerm(toks[2:end])
	if err != nil {
		return nil, err
	}

	if end != len(toks) {
		// we have a representation
		if toks[len(toks)-1] == "representation" {
			return nil, fmt.Errorf("map 'representation' keyword must be followed by a representation type")
		}

		reprType := toks[end+1]
		switch reprType {
		case "map":
			if len(toks) > end+2 {
				return nil, fmt.Errorf("extraneous tokens found after map 'map' representation declaration")
			}
			mt.Representation = &MapRepresentation{Map: &MapRepresentation_Map{}}
		case "stringpairs":
			innerDelim, entryDelim, err := parseStringPairsRepresentation(s)
			if err != nil {
				return nil, err
			}
			repr := &Representation_StringPairs{InnerDelim: innerDelim, EntryDelim: entryDelim}
			mt.Representation = &MapRepresentation{StringPairs: repr}
		case "listpairs":
			if len(toks) > end+2 {
				return nil, fmt.Errorf("extraneous tokens found after map 'listpairs' representation declaration")
			}
			mt.Representation = &MapRepresentation{ListPairs: &Representation_ListPairs{}}
		case "advanced":
			if len(toks) != end+3 {
				return nil, fmt.Errorf("malformed map 'advanced' representation declaration")
			}
			adlName := AdvancedDataLayoutName(toks[end+2])
			mt.Representation = &MapRepresentation{Advanced: &adlName}
		default:
			return nil, fmt.Errorf("unknown map 'representation' type [%v]", reprType)
		}
	}

	return mt, nil
}

func parseMapTypeTerm(toks []string) (*TypeMap, error) {
	if len(toks) < 5 {
		// not a great error message, should more clearly tell user
		// what is actually missing
		return nil, fmt.Errorf("map TypeTerms must be at least 5 tokens")
	}

	if toks[len(toks)-1] != "}" {
		return nil, fmt.Errorf("map TypeTerm must end with matching '}'")
	}

	keyType := toks[1]
	if toks[2] != ":" {
		return nil, fmt.Errorf("expected ':' between map key type and value type")
	}

	valTermStart := 3
	var nullable bool
	if toks[3] == "nullable" {
		nullable = true
		valTermStart++
	}

	valueType, err := parseTypeTerm(toks[valTermStart : len(toks)-1])
	if err != nil {
		return nil, err
	}

	return &TypeMap{
		Kind:          "map",
		KeyType:       keyType,
		ValueType:     valueType,
		ValueNullable: nullable,
	}, nil
}

func parseListType(toks []string, s *bufio.Scanner) (*TypeList, error) {
	end := len(toks)
	if end >= 6 && toks[5] == "representation" {
		end = 5
	} else if end >= 7 && toks[6] == "representation" { // could have a "nullable"
		end = 6
	}

	lt, err := parseListTypeTerm(toks[2:end])
	if err != nil {
		return nil, err
	}

	if end != len(toks) {
		// we have a representation
		if toks[len(toks)-1] == "representation" {
			return nil, fmt.Errorf("map 'representation' keyword must be followed by a representation type")
		}

		reprType := toks[end+1]
		if reprType == "advanced" {
			if len(toks) != end+3 {
				return nil, fmt.Errorf("malformed map 'advanced' representation declaration")
			}
			adlName := AdvancedDataLayoutName(toks[end+2])
			lt.Representation = &ListRepresentation{Advanced: &adlName}
		} else {
			return nil, fmt.Errorf("unknown map 'representation' type [%v]", reprType)
		}
	}

	return lt, nil
}

func parseListTypeTerm(toks []string) (*TypeList, error) {
	toks = toks[1:]

	last := toks[len(toks)-1]
	if last != "]" {
		return nil, fmt.Errorf("list TypeTerm must end with matching ']'")
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
		Kind:          "list",
		ValueType:     subtype,
		ValueNullable: nullable,
	}, nil
}

func tokenToLink(tok string) *TypeLink {
	linktype := NamedType(tok[1:])

	if linktype == "Any" {
		return &TypeLink{Kind: "link"}
	}

	return &TypeLink{
		Kind:         "link",
		ExpectedType: linktype,
	}
}

func parseStringPairsRepresentation(s *bufio.Scanner) (innerDelim string, entryDelim string, err error) {
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				return "", "", fmt.Errorf("extraneous tokens found at end of stringpairs representation block [%v]", toks[1:])
			}
			if innerDelim == "" {
				return "", "", fmt.Errorf("no 'innerDelim' in stringpairs representation")
			}
			if entryDelim == "" {
				return "", "", fmt.Errorf("no 'entryDelim' in stringpairs representation")
			}
			return innerDelim, entryDelim, nil
		}

		if len(toks) != 2 {
			return "", "", fmt.Errorf("invalid tokens found in stringpairs representation block [%v]", toks)
		}

		switch toks[0] {
		case "innerDelim":
			if innerDelim != "" {
				return "", "", fmt.Errorf("multiple 'innerDelim's in stringpairs representation")
			}
			innerDelim = toks[1]
		case "entryDelim":
			if entryDelim != "" {
				return "", "", fmt.Errorf("multiple 'entryDelim's in stringpairs representation")
			}
			entryDelim = toks[1]
		default:
			return "", "", fmt.Errorf("unrecognized token '%q' in stringpairs representation", toks[0])
		}
	}

	return "", "", fmt.Errorf("reached end of file while parsing stringpairs representation")
}

func parseStringJoinRepresentation(s *bufio.Scanner) (join string, fieldOrder []string, err error) {
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				return "", nil, fmt.Errorf("extraneous tokens found at end of stringjoin representation block [%v]", toks[1:])
			}
			if join == "" {
				return "", nil, fmt.Errorf("no 'join' in stringjoin representation")
			}
			return join, fieldOrder, nil
		}

		switch toks[0] {
		case "join":
			if join != "" {
				return "", nil, fmt.Errorf("multiple 'join's in stringjoin representation")
			}
			if len(toks) != 2 {
				return "", nil, fmt.Errorf("invalid tokens found in stringjoin representation block [%v]", toks)
			}
			join = toks[1]
		case "fieldOrder":
			fieldOrder, err = parseFieldOrder(toks)
			if err != nil {
				return "", nil, err
			}
		default:
			return "", nil, fmt.Errorf("unrecognized token '%q' in stringjoin representation", toks[0])
		}
	}

	return "", nil, fmt.Errorf("reached end of file while parsing stringjoin representation")
}

func parseTupleRepresentation(s *bufio.Scanner) (fieldOrder []string, err error) {
	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				return nil, fmt.Errorf("extraneous tokens found at end of tuple representation block [%v]", toks[1:])
			}
			if len(fieldOrder) == 0 {
				return nil, fmt.Errorf("no 'fieldOrder' in tuple representation")
			}
			return fieldOrder, nil
		}

		if toks[0] == "fieldOrder" {
			fieldOrder, err = parseFieldOrder(toks)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unrecognized token '%q' in tuple representation", toks[0])
		}
	}

	return nil, fmt.Errorf("reached end of file while parsing tuple representation")
}

func parseFieldOrder(toks []string) ([]string, error) {
	if toks[1] != "[" {
		return nil, fmt.Errorf("expected opening '[' in tuple representation, found '%s'", toks[1])
	}
	if toks[len(toks)-1] != "]" {
		return nil, fmt.Errorf("expected closing ']' in tuple representation, found '%s'", toks[len(toks)-1])
	}
	// assume comma separated list between
	return strings.Split(strings.Join(toks[2:len(toks)-1], ""), ","), nil
}

func parseStructRepr(line []string, s *bufio.Scanner, freps map[string]SRMFieldDetails) (*StructRepresentation, error) {
	if len(line) < 3 {
		return nil, fmt.Errorf("no representation kind given")
	}

	reprkind := line[2]

	if reprkind != "map" && len(freps) > 0 {
		return nil, fmt.Errorf("%s struct representation cannot have field details", reprkind)
	}

	switch reprkind {
	case "map":
		if len(line) > 3 {
			return nil, fmt.Errorf("unexpected tokens after 'representation map'")
		}
		return &StructRepresentation{Map: &StructRepresentation_Map{Fields: freps}}, nil
	case "tuple":
		repr := &StructRepresentation_Tuple{}
		if len(line) > 3 {
			fieldOrder, err := parseTupleRepresentation(s)
			if err != nil {
				return nil, err
			}
			repr.FieldOrder = fieldOrder
			// TODO: check fields in fieldOrder are in the list of struct fields
		}
		return &StructRepresentation{Tuple: repr}, nil
	case "stringpairs":
		innerDelim, entryDelim, err := parseStringPairsRepresentation(s)
		if err != nil {
			return nil, err
		}
		repr := &Representation_StringPairs{InnerDelim: innerDelim, EntryDelim: entryDelim}
		return &StructRepresentation{StringPairs: repr}, nil
	case "stringjoin":
		join, fieldOrder, err := parseStringJoinRepresentation(s)
		if err != nil {
			return nil, err
		}
		repr := StructRepresentation_StringJoin{Join: join}
		if len(fieldOrder) > 0 {
			repr.FieldOrder = fieldOrder
		}
		return &StructRepresentation{StringJoin: &repr}, nil
	case "listpairs":
		if len(line) > 3 {
			return nil, fmt.Errorf("unexpected tokens after 'representation listpairs'")
		}
		return &StructRepresentation{ListPairs: &Representation_ListPairs{}}, nil
	default:
		return nil, fmt.Errorf("unrecognized struct representation: %s", reprkind)
	}
}

func ParseSchema(s *bufio.Scanner) (*Schema, error) {
	schema := &Schema{TypesMap: make(TypesMap)}

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
			schema.TypesMap[name] = t
		case "advanced":
			if len(toks) == 1 {
				return nil, fmt.Errorf("'advanced' declaration requires a name token")
			}
			if len(toks) != 2 {
				return nil, fmt.Errorf("extraneous tokens after 'advanced' declaration")
			}
			if schema.AdvancedMap == nil {
				schema.AdvancedMap = make(AdvancedMap)
			}
			schema.AdvancedMap[toks[1]] = Advanced{"advanced"}
		default:
			return nil, fmt.Errorf("unexpected token: %q", toks[0])
		}
	}
	return schema, nil
}
