package test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/whyrusleeping/ipld-schema/parser"
)

const fixturesDir = "./fixtures/bulk/"

type Fixture struct {
	Schema          string
	Expected        string
	ExpectedParsed  interface{}
	Blocks          []FixtureBlock
	BadBlocks       []string `yaml:"badBlocks"`
	BadBlocksParsed []interface{}
}

type FixtureBlock struct {
	Actual         string
	ActualParsed   interface{}
	Expected       string
	ExpectedParsed interface{}
}

func TestBulk(t *testing.T) {
	files, err := ioutil.ReadDir(fixturesDir)
	assert.NoError(t, err)

	for _, f := range files {
		verifyFixture(t, f.Name())
	}
}

func verifyFixture(t *testing.T, name string) {
	fmt.Printf("verifyFixture(%s)\n", name)

	fx := loadFixture(t, name)

	parsedSchema, err := parser.ParseSchema(bufio.NewScanner(strings.NewReader(fx.Schema)))
	assert.NoError(t, err)

	actual, err := json.MarshalIndent(parsedSchema, "", "  ")
	assert.NoError(t, err)
	expected, err := json.MarshalIndent(fx.ExpectedParsed, "", "  ")
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))

	/*
		var out bytes.Buffer
		err = schema.ExportIpldSchema(parsedSchema, &out)
		assert.NoError(t, err)
		regenerated := strings.ReplaceAll(out.String(), "\t", "  ")
		regenerated = regenerated[0 : len(regenerated)-1]
	*/

	/* TODO: order of map[] fields in some structs causes shuffling so we can't do a straight compare
		without imposing order retention
	fmt.Println("--------- Original Schema:")
	fmt.Printf(fx.Schema)
	fmt.Println("--------- Regenerated Schema:")
	fmt.Printf(regenerated)

	assert.Equal(t, fx.Schema, regenerated)
	*/
}

func loadFixture(t *testing.T, name string) Fixture {
	file, err := ioutil.ReadFile(fixturesDir + name)
	assert.NoError(t, err)

	var fx Fixture
	err = yaml.Unmarshal(file, &fx)
	assert.NoError(t, err)

	err = json.Unmarshal([]byte(fx.Expected), &fx.ExpectedParsed)
	assert.NoError(t, err)

	for _, block := range fx.Blocks {
		err = json.Unmarshal([]byte(block.Actual), &block.ActualParsed)
		assert.NoError(t, err)
		err = json.Unmarshal([]byte(block.Expected), &block.ExpectedParsed)
		assert.NoError(t, err)
	}

	fx.BadBlocksParsed = make([]interface{}, len(fx.BadBlocks))
	for i, block := range fx.BadBlocks {
		err = json.Unmarshal([]byte(block), &(fx.BadBlocksParsed[i]))
		assert.NoError(t, err)
	}

	return fx
}
