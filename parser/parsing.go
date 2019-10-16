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

func parseType(tline []string, s *bufio.Scanner) (Type, error) {
	if len(tline) < 3 {
		return nil, fmt.Errorf("expected at least three tokens on type definition line")
	}

	// thinking we should just call 'parseTypeTerm' here...

	tname := tline[1]
	ttype := tline[2]

	var t Type
	var err error
	switch ttype {
	case "struct":
		if len(tline) < 4 || tline[3] != "{" {
			return nil, fmt.Errorf("struct declaration must contain an open brace")
		}

		if len(tline) > 4 {
			// parse fucky struct declaration
			if tline[len(tline)-1] != "}" {
				return nil, fmt.Errorf("oneline struct declaration must terminate on same line")
			}

			inner := tline[4 : len(tline)-1]
			if len(inner) == 0 {
				ts := NewTypeStruct(tname, &StructRepresentation{Map: &StructRepresentation_Map{}})
				return &ts, nil
			}

			strf, err := parseStructField(inner)
			if err != nil {
				return nil, err
			}

			ts := NewTypeStruct(tname, &StructRepresentation{Map: &StructRepresentation_Map{}})
			ts.Fields.Append(*strf)
			return &ts, nil
		}

		t, err = parseStruct(tname, s)
	case "union":
		if len(tline) != 4 || tline[3] != "{" {
			return nil, fmt.Errorf("union declaration must end in an open brace")
		}
		t, err = parseUnion(tname, s)
	case "enum":
		if len(tline) != 4 || tline[3] != "{" {
			return nil, fmt.Errorf("enum declaration must end in an open brace")
		}
		t, err = parseEnum(tname, s)
	case "bool", "bytes", "float", "int", "link", "string", "null":
		if ttype == "bytes" && len(tline) >= 4 && tline[3] == "representation" {
			if len(tline) != 6 || tline[4] != "advanced" {
				return nil, fmt.Errorf("%s declaration has malformed 'advanced' representation", ttype)
			}

			adlName := AdvancedDataLayoutName(tline[5])
			tb := NewTypeBytes(tname, &BytesRepresentation{Advanced: &adlName})
			return &tb, nil
		}

		if len(tline) == 3 {
			return NewSimpleType(tname, ttype), nil
		}

		return nil, fmt.Errorf("%s declaration cannot be followed by additional tokens", ttype)
	case "{":
		t, err = parseMapType(tname, tline, s)
		if err != nil {
			return nil, err
		}
	case "[":
		t, err = parseListType(tname, tline, s)
		if err != nil {
			return nil, err
		}
	case "=":
		if len(tline) != 4 {
			return nil, fmt.Errorf("%s copy type declaration requires a fromType type name", tname)
		}
		tc := NewTypeCopy(tname, tline[3])
		return &tc, nil
	default:
		t, err = parseTypeTerm(tname, tline[2:])
	}

	return t, err
}

func parseEnum(name string, s *bufio.Scanner) (*TypeEnum, error) {
	vals := make([]EnumValue, 0)
	reprVals := make([]*string, 0)
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

			vals = append(vals, ev)

			if ntoks == 5 {
				if toks[2] != "(" && toks[4] != ")" {
					return nil, fmt.Errorf("unexpected tokens after EnumValue, expected representation value")
				}

				reprVals = append(reprVals, &toks[3])
			} else {
				reprVals = append(reprVals, nil)
			}

			continue
		}

		if toks[0] == "}" {
			er := EnumRepresentation{}

			if ntoks == 3 && toks[1] == "representation" && (toks[2] == "int" || toks[2] == "string") {
				if toks[2] == "int" {
					eri := EnumRepresentation_Int{}
					for i, v := range vals {
						if reprVals[i] != nil {
							i, err := strconv.ParseInt(*reprVals[i], 10, 64)
							if err != nil {
								return nil, fmt.Errorf("'int' union representation may only use int values (%v)", v)
							}
							eri.AddMapping(v, int(i))
						}
					}
					er.Int = &eri
				}
			} else if ntoks != 1 {
				return nil, fmt.Errorf("unexpected tokens after end of enum")
			}

			if er.Int == nil {
				ers := EnumRepresentation_String{}
				for i, v := range vals {
					if reprVals[i] != nil {
						ers.AddMapping(v, *reprVals[i])
					}
				}
				er.String = &ers
			}

			te := NewTypeEnum(name, &er)
			for _, ev := range vals {
				te.Members.Append(ev)
			}
			return &te, nil
		}

		return nil, fmt.Errorf("unexpected token: %s", toks[0])
	}

	return nil, fmt.Errorf("unterminated enum")
}

func parseUnion(name string, s *bufio.Scanner) (*TypeUnion, error) {
	type unionVal struct {
		key string
		typ Type
	}
	unionVals := make([]unionVal, 0)
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
				unionVals = append(unionVals, unionVal{key, tokenToLink("", toks[1])}) // anonymous link
			} else {
				// TODO: validate characters
				unionVals = append(unionVals, unionVal{key, NewNamedType(toks[1])})
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
				rep := UnionRepresentation_Kinded{}
				for _, k := range unionVals {
					rep.AddMapping(RepresentationKind(k.key), k.typ)
				}
				repr := &UnionRepresentation{Kinded: &rep}
				tu := NewTypeUnion(name, repr)
				return &tu, nil
			case "inline":
				if len(toks) < 4 {
					return nil, fmt.Errorf("expected open bracket for inline union representation block")
				}
				urep, err := parseUnionInlineRepresentation(s)
				if err != nil {
					return nil, err
				}

				for _, k := range unionVals {
					urep.DiscriminantTable.AddMapping(k.key, k.typ)
				}

				repr := &UnionRepresentation{Inline: urep}
				tu := NewTypeUnion(name, repr)
				return &tu, nil
			case "keyed":
				rep := UnionRepresentation_Keyed{}
				for _, k := range unionVals {
					rep.AddMapping(k.key, k.typ)
				}

				repr := &UnionRepresentation{Keyed: &rep}
				tu := NewTypeUnion(name, repr)
				return &tu, nil
			case "envelope":
				if len(toks) < 4 {
					return nil, fmt.Errorf("expected open bracket for envelope union representation block")
				}
				urep, err := parseUnionEnvelopeRepresentation(s)
				if err != nil {
					return nil, err
				}

				for _, k := range unionVals {
					urep.DiscriminantTable.AddMapping(k.key, k.typ)
				}

				repr := &UnionRepresentation{Envelope: urep}
				tu := NewTypeUnion(name, repr)
				return &tu, nil
			case "byteprefix":
				rep := UnionRepresentation_BytePrefix{}
				for _, k := range unionVals {
					nt, ok := k.typ.(NamedType)
					if !ok {
						return nil, fmt.Errorf("'byteprefix' union representation may only contain named types (%v)", k.typ)
					}
					i, err := strconv.ParseInt(string(k.key), 10, 64)
					if err != nil {
						return nil, fmt.Errorf("'byteprefix' union representation may only use int discriminators (%v)", k.typ)
					}
					rep.AddMapping(nt, int(i))
				}

				repr := &UnionRepresentation{BytePrefix: &rep}
				tu := NewTypeUnion(name, repr)
				return &tu, nil
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

func parseStruct(name string, s *bufio.Scanner) (*TypeStruct, error) {
	st := NewTypeStruct(name, nil)
	maprep := StructRepresentation_Map{}

	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		if toks[0] == "}" {
			if len(toks) > 1 {
				if toks[1] == "representation" {
					repr, err := parseStructRepr(toks, s, &maprep)
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
				st.Representation = &StructRepresentation{Map: &maprep}
			}

			return &st, nil
		}

		var srmFieldTokens []string
		typeTermFieldTokens := toks

		var frep *StructRepresentation_Map_FieldDetails
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

		strf, err := parseStructField(typeTermFieldTokens)
		if err != nil {
			return nil, err
		}

		if len(srmFieldTokens) > 0 {
			var err error
			frep, err = parseStructFieldRep(strf.GetName(), strf.Type, srmFieldTokens)
			if err != nil {
				return nil, err
			}
		}

		st.Fields.Append(*strf)
		if frep != nil {
			maprep.AddFieldDetails(*frep)
		}
	}

	return &st, nil
}

func parseStructField(toks []string) (*StructField, error) {
	var optional, nullable bool
	var i int = 1

loop:
	for ; i < len(toks)-1; i++ {
		switch toks[i] {
		case "optional":
			if optional {
				return nil, fmt.Errorf("multiple optional keywords")
			}
			optional = true
		case "nullable":
			if nullable {
				return nil, fmt.Errorf("multiple nullable keywords")
			}
			nullable = true
		default:
			break loop
		}
	}

	trepr, err := parseTypeTerm("", toks[i:]) // "" for anonymous type
	if err != nil {
		return nil, err
	}

	fname := toks[0]

	sf := NewStructField(fname, nullable, optional, trepr)
	return &sf, nil
}

func parseStructFieldRep(fieldName string, typeTerm TypeTerm, toks []string) (*StructRepresentation_Map_FieldDetails, error) {
	var implicit interface{}
	var rename string

	applyInfo := func(k, v string) error {
		switch k {
		case "implicit":
			if implicit != nil {
				return fmt.Errorf("duplicate implicit in struct field representation")
			}
			cv, err := coerceImplicitValueType(typeTerm, v)
			if err != nil {
				return err
			}
			implicit = cv
		case "rename":
			if rename != "" {
				return fmt.Errorf("duplicate rename in struct field representation")
			}
			rename = v
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

	srmfd := NewStructRepresentation_Map_FieldDetails(fieldName, implicit, rename)
	return &srmfd, nil
}

func coerceImplicitValueType(typeTerm TypeTerm, value string) (interface{}, error) {
	nt, ok := typeTerm.(NamedType)
	if !ok {
		return nil, fmt.Errorf("Cannot accept implicit values for complex types [%v]", typeTerm)
	}

	switch nt.GetName() {
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

func parseTypeTerm(name string, toks []string) (Type, error) {
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

		return tokenToLink(name, toks[0]), nil
	}

	// handle anonymous [] and {} types here
	switch toks[0] {
	case "[":
		return parseListTypeTerm(name, toks)
	case "{":
		return parseMapTypeTerm(name, toks)
	default:
		if len(toks) == 1 {
			return NewNamedType(toks[0]), nil
		}
		// TODO: better error
		fmt.Println("failed to parse token: ", toks[0])
		fmt.Println("full line: ")
		fmt.Println(toks)
		panic("Can't deal")
	}
}

func parseMapType(name string, toks []string, s *bufio.Scanner) (*TypeMap, error) {
	end := len(toks)
	if end >= 8 && toks[7] == "representation" {
		end = 7
	} else if end >= 9 && toks[8] == "representation" { // could have a "nullable"
		end = 8
	}

	mt, err := parseMapTypeTerm(name, toks[2:end])
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

func parseMapTypeTerm(name string, toks []string) (*TypeMap, error) {
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

	valueType, err := parseTypeTerm("", toks[valTermStart:len(toks)-1]) // "" for anonymous type
	if err != nil {
		return nil, err
	}

	tm := NewTypeMap(name, keyType, valueType, nullable, nil)
	return &tm, nil
}

func parseListType(name string, toks []string, s *bufio.Scanner) (*TypeList, error) {
	end := len(toks)
	if end >= 6 && toks[5] == "representation" {
		end = 5
	} else if end >= 7 && toks[6] == "representation" { // could have a "nullable"
		end = 6
	}

	lt, err := parseListTypeTerm(name, toks[2:end])
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

func parseListTypeTerm(name string, toks []string) (*TypeList, error) {
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

	subtype, err := parseTypeTerm("", toks) // "" for anonymous type
	if err != nil {
		return nil, err
	}

	tl := NewTypeList(name, subtype, nullable, nil)
	return &tl, nil
}

func tokenToLink(name string, tok string) *TypeLink {
	linkType := NewNamedType(tok[1:])

	tl := NewTypeLink(name, linkType)
	return &tl
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

func parseStructRepr(line []string, s *bufio.Scanner, maprep *StructRepresentation_Map) (*StructRepresentation, error) {
	if len(line) < 3 {
		return nil, fmt.Errorf("no representation kind given")
	}

	reprkind := line[2]

	if reprkind != "map" && maprep.FieldDetailsCount() > 0 {
		return nil, fmt.Errorf("'%s' struct representation cannot have field details", reprkind)
	}

	switch reprkind {
	case "map":
		if len(line) > 3 {
			return nil, fmt.Errorf("unexpected tokens after 'representation map'")
		}
		return &StructRepresentation{Map: maprep}, nil
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
	schema := &Schema{TypesList: &TypesList{}}

	for s.Scan() {
		toks := tokens(s.Text())
		if len(toks) == 0 {
			continue
		}

		switch toks[0] {
		case "type":
			t, err := parseType(toks, s)
			if err != nil {
				fmt.Println("failed to parse line: ")
				fmt.Println(s.Text())
				fmt.Printf("%q\n", toks)
				return nil, err
			}
			schema.TypesList.Append(t)
		case "advanced":
			if len(toks) == 1 {
				return nil, fmt.Errorf("'advanced' declaration requires a name token")
			}
			if len(toks) != 2 {
				return nil, fmt.Errorf("extraneous tokens after 'advanced' declaration")
			}
			if schema.AdvancedList == nil {
				schema.AdvancedList = &AdvancedList{}
			}
			schema.AdvancedList.Append(NewAdvanced(toks[1]))
		default:
			return nil, fmt.Errorf("unexpected token: %q", toks[0])
		}
	}
	return schema, nil
}
