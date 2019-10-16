package test

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/ipld/go-ipld-schema/parser"
	"github.com/stretchr/testify/assert"
)

func TestSchemaSchema(t *testing.T) {
	expectedString, expectedParsed := loadExpected(t)
	actualString, actualParsed := loadActual(t)

	for name, typ := range expectedParsed {
		assert.Contains(t, actualParsed, name)
		assert.Equal(t, typ, actualParsed[name], name)
	}

	assert.Equal(t, strings.TrimSpace(expectedString), actualString)
}

func loadExpected(t *testing.T) (string, map[string]interface{}) {
	var expected map[string]interface{}

	file, err := ioutil.ReadFile("./fixtures/schema-schema.ipldsch.json")
	assert.NoError(t, err)

	err = json.Unmarshal(file, &expected)
	assert.NoError(t, err)

	return string(file), expected
}

func loadActual(t *testing.T) (string, map[string]interface{}) {
	fi, err := os.Open("./fixtures/schema-schema.ipldsch")
	assert.NoError(t, err)
	defer fi.Close()
	parsedSchema, err := parser.ParseSchema(bufio.NewScanner(fi))
	assert.NoError(t, err)

	jsonified, err := json.MarshalIndent(parsedSchema, "", "\t")
	assert.NoError(t, err)

	var actual map[string]interface{}
	err = json.Unmarshal(jsonified, &actual)
	assert.NoError(t, err)

	return string(jsonified), actual
}
