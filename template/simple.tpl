package schame 

import (
	"database/sql"
	"time"
)

type {{.Name}} struct {
{{range .Columns}}
	{{.Name}} {{.Type}}
{{- end}}
}

func {{.Name}}Fields() []string {
	return "{{.SelectFields}}"
}

func Scan{{.Name}}(row *sql.Row) (*{{.Name}}, error) {
	r := {{.Name}}{}
	err := row.Scan({{.ScanFields}})
	return &r, err
}

func Scan{{.Name}}s(rows *sql.Rows) ([]*{{.Name}}, error) {
	rs := []*{{.Name}}{}
	for rows.Next() {
		r := {{.Name}}{}
		err := rows.Scan({{.ScanFields}})
		if err != nil {
			return nil, err
		}
		rs = append(rs, &r)
	}
	return rs, nil
}
