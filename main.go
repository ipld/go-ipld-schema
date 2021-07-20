package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli"

	parser "github.com/ipld/go-ipld-schema/parser"
	schema "github.com/ipld/go-ipld-schema/schema"
)

var schemaToJsonCmd = cli.Command{
	Name: "to-json",
	Action: func(c *cli.Context) error {
		if !c.Args().Present() {
			return fmt.Errorf("must specify schema file to read")
		}

		fi, err := os.Open(c.Args().First())
		if err != nil {
			return err
		}
		defer fi.Close()

		s := bufio.NewScanner(fi)
		sc, err := parser.ParseSchema(s)
		if err != nil {
			return err
		}

		// JSON schema types should be nested within a "schema" key
		out, err := json.MarshalIndent(sc, "", "\t")
		if err != nil {
			panic(err)
		}

		fmt.Println(string(out))

		return nil
	},
}

var schemaToSchemaCmd = cli.Command{
	Name: "to-schema",
	Action: func(c *cli.Context) error {
		if !c.Args().Present() {
			return fmt.Errorf("must specify schema file to read")
		}

		fi, err := os.Open(c.Args().First())
		if err != nil {
			return err
		}
		defer fi.Close()

		s := bufio.NewScanner(fi)
		sch, err := parser.ParseSchema(s)
		if err != nil {
			return err
		}

		if err := schema.PrintSchema(sch, os.Stdout); err != nil {
			panic(err)
		}

		return nil
	},
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		schemaToJsonCmd,
		schemaToSchemaCmd,
	}

	app.Run(os.Args)
}
