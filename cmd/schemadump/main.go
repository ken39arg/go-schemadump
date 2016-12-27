package main

import (
	"flag"

	schemadump "github.com/ken39arg/go-schemadump"
)

var (
	dsn      = flag.String("dsn", "", "target data source name (see github.com/go-sql-driver/mysql)")
	schema   = flag.String("schema", "", "path to schema file")
	output   = flag.String("output", "STDOUT", `output file puttern. if separate by table using '%t' path/to/table/%t.auto.go`)
	template = flag.String("template", "", "table template path")
)

func main() {
	flag.Parse()
	tables := flag.Args()
	dumper := &schemadump.Dumper{
		Output:   *output,
		Template: *template,
		DSN:      *dsn,
		Schema:   *schema,
		Tables:   tables,
	}
	dumper.Run()
}
