# schemadump

make go template from mysql schema

## Install

```
$ go get github.com/ken39arg/go-schemadump/cmd/schemadump
```

## Usage

```
schemadump -help
```

###  from mysql dsn

```
schemadump -dsn "user:pass@tcp/dbname"
```

DSN is https://github.com/go-sql-driver/mysql

### from SQL Schame

```
schemadump -schema "pass/to/schame.sql"
```

### output to file

```
schemadump -schema "pass/to/schame.sql" -output "path/to/schema/%t.auto.go"
```

### use template

```
schemadump -schema "pass/to/schame.sql" -output "path/to/schema/%t.auto.go" -template "path/to/template.tpl"
```

template is https://golang.org/pkg/text/template

#### Template params

```
// . = schamedump.Table

type Table struct {
	Name              string
	DBName            string // original table name
	Columns           []Column
	Indexes           []Index
	PrimaryKey        Index
	NonPrimaryIndexes []Index
	ColumnDBNames     []string
	SelectFields      string
	ScanFields        string
}

type Column struct {
	Name          string // UpperCamel Name  struct
	Type          string
	Nullable      bool
	Size          uint32
	Default       string
	AutoIncrement bool
	Extra         string
	DBName        string // original column_name
	DBType        string // original type (show columns from *)
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Primary bool
}

```

* sample = https://github.com/ken39arg/go-schemadump/tree/master/template
