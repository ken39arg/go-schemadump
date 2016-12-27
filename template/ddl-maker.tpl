package schema

import (
    "database/sql"
    "time"

    "github.com/kayac/ddl-maker/dialect"
    "github.com/kayac/ddl-maker/dialect/mysql"
)

type {{.Name}} struct {
{{range .Columns}}
    {{- $hasSize := lt 0 .Size -}}
    {{- $hasDef := ne .Default "" -}}
	{{.Name}} {{.Type}}  {{if or .Nullable .AutoIncrement $hasSize  $hasDef -}}
        `ddl:"{{if  $hasSize}}size={{.Size}},{{end}}{{if .Nullable }}null,{{end}}{{if $hasDef }}default={{.Default}},{{end}}{{if .AutoIncrement }}auto{{end}}"`
    {{end}}
{{- end}}
}

{{if .PrimaryKey.Primary }}
func (c {{.Name}}) PrimaryKey() dialect.PrimaryKey {
    return mysql.AddPrimaryKey(
    {{- range $i, $c := .PrimaryKey.Columns}}
        {{- if lt 0 $i}},{{end}}"{{.}}"
    {{- end -}}
    )
}
{{- end}}

{{if .NonPrimaryIndexes }}
func (c {{.Name}}) Indexes() dialect.Indexes {
    return dialect.Indexes{
        {{- range .NonPrimaryIndexes}}
            mysql.{{if .Unique }}AddUniqueIndex{{else}}AddIndex{{end}}("{{.Name}}"{{range .Columns}},"{{.}}"{{end}}),
        {{- end}}
    }
}
{{- end}}

