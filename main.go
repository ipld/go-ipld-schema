package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli"

	"github.com/whyrusleeping/ipld-schema/gen-go"
	"github.com/whyrusleeping/ipld-schema/parser"
	"github.com/whyrusleeping/ipld-schema/schema"
)

var genGoCmd = cli.Command{
	Name: "gen-go",
	Action: func(c *cli.Context) error {
		if !c.Args().Present() {
			return fmt.Errorf("must specify schema file to generate code for")
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

		if err := gengo.GolangCodeGen(sc, os.Stdout); err != nil {
			return err
		}

		return nil
	},
}

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

		out, err := json.MarshalIndent(sc, "", "  ")
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
		sc, err := parser.ParseSchema(s)
		if err != nil {
			return err
		}

		if err := schema.ExportIpldSchema(sc, os.Stdout); err != nil {
			panic(err)
		}

		return nil
	},
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		genGoCmd,
		schemaToJsonCmd,
		schemaToSchemaCmd,
	}

	app.RunAndExitOnError()
}
