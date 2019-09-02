package test

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whyrusleeping/ipld-schema/parser"
)

func TestSchemaSchema(t *testing.T) {
	expected := loadExpected(t)
	actual := loadActual(t)

	for name, typ := range expected {
		assert.Contains(t, actual, name)
		assert.Equal(t, typ, actual[name], name)
	}
}

func loadExpected(t *testing.T) map[string]interface{} {
	var expected map[string]map[string]interface{}

	file, err := ioutil.ReadFile("./fixtures/schema-schema.ipldsch.json")
	assert.NoError(t, err)

	err = json.Unmarshal(file, &expected)
	assert.NoError(t, err)

	return expected["schema"]
}

func loadActual(t *testing.T) map[string]interface{} {
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

	return actual
}
