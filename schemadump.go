package schemadump

import (
	"bytes"
	"database/sql"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	mysqltest "github.com/lestrrat/go-test-mysqld"
)

type Dumper struct {
	Output   string
	Template string
	DSN      string
	Schema   string
	Tables   []string
	Package  string
}

func (d *Dumper) Run() {
	var mysqld *mysqltest.TestMysqld
	if d.DSN == "" {
		var err error
		mysqld, err = mysqltest.NewMysqld(nil)
		if err != nil {
			log.Panicf("Failed to start mysqld: %s", err)
			return
		}
		defer mysqld.Stop()
		d.DSN = mysqld.Datasource("test", "", "", 0)
	}

	db, err := sql.Open("mysql", d.DSN)
	if err != nil {
		log.Panicf("Failed to Open mysqld: %s", err)
		return
	}
	defer db.Close()

	if d.Schema != "" {
		log.Printf("Load schema " + d.Schema)
		queries := parseSchameFile(d.Schema)
		for _, query := range queries {
			_, err := db.Exec(query)
			if err != nil && !strings.Contains(err.Error(), "Query was empty") {
				log.Printf("load err: %s", err)
			}
		}
	}

	if d.Package == "" {
		d.Package = "scheama"
	}

	inspecter := &Inspector{db: db}
	if 0 < len(d.Tables) {
		inspecter.InspectTables(d.Tables...)
	} else {
		inspecter.Inspect()
	}

	d.OutputTables(inspecter)
}

func (d *Dumper) OutputTables(inspecter *Inspector) {
	var tpl *template.Template
	const structTemplate = `
type {{.Name}} struct {
{{range .Columns}}
	{{.Name}} {{.Type}}
{{- end}}
}
`
	if d.Template == "" {
		tpl = template.Must(template.New("struct").Parse(structTemplate))
	} else {
		tpl = template.Must(template.ParseFiles(d.Template))
	}

	var io *os.File
	separateFile := false
	if d.Output == "STDOUT" || d.Output == "" {
		log.Printf("Outpu is STDOUT")
		io = os.Stdout
	} else if -1 == strings.Index(d.Output, "%t") {
		var err error
		io, err = os.Create(d.Output)
		if err != nil {
			log.Panicf("Can't create new file: %s, %s", d.Output, err)
		}
	} else {
		separateFile = true
	}

	buf := &bytes.Buffer{}
	for _, table := range inspecter.Tables {
		log.Printf("table found " + table.Name)
		err := tpl.Execute(buf, table)
		if err != nil {
			log.Panicf("executing error: table: %s, err: %s", table.Name, err)
			continue
		}
		if separateFile {
			path := strings.Replace(d.Output, "%t", table.DBName, -1)
			f, err := os.Create(path)
			if err != nil {
				log.Panicf("Can't create new file: %s, %s", path, err)
			}
			writeBuffer(f, buf)
			buf.Reset()
			f.Close()
		}
	}
	if !separateFile {
		writeBuffer(io, buf)
		io.Close()
	}
}

func parseSchameFile(schema string) []string {
	data, err := ioutil.ReadFile(schema)
	if err != nil {
		log.Panicf("Failed to Open schema %s: %s", schema, err)
	}

	queries := []string{}
	for _, stmt := range strings.Split(string(data), ";") {
		queries = append(queries, stmt)
	}
	return queries
}

func writeBuffer(io *os.File, buf *bytes.Buffer) {
	data, err := format.Source(buf.Bytes())
	if err != nil {
		log.Panicf("go fmt err: %s \n%v", err, buf)
	}
	_, err = io.Write(data)
	if err != nil {
		log.Panicf("write errror: %s", err)
	}
}
