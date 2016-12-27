package schemadump

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/serenize/snaker"
)

type Column struct {
	Name     string // UpperCamel Name  struct
	Type     string
	Nullable bool
	Size     uint32
	Default  string
	Extra    string
	DBName   string // original column_name
	DBType   string // original type (show columns from *)
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Primary bool
}

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

type Inspector struct {
	db     *sql.DB
	Tables []Table
}

func NewInspector(db *sql.DB) *Inspector {
	ins := &Inspector{db: db}
	ins.Inspect()
	return ins
}

func (ins *Inspector) Inspect() {
	rows, err := ins.db.Query("SHOW TABLES")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			log.Panic(err)
		}
		t := ins.inspectTable(table)
		ins.Tables = append(ins.Tables, t)
	}
}

func (ins *Inspector) InspectTables(tables ...string) {
	for _, table := range tables {
		rows, err := ins.db.Query(fmt.Sprintf("SHOW TABLES LIKE '%s'", table))
		if err != nil {
			log.Panic(err)
		}
		if rows.Next() {
			t := ins.inspectTable(table)
			ins.Tables = append(ins.Tables, t)
		}
		rows.Close()
	}
}

func (ins *Inspector) inspectTable(name string) Table {
	t := Table{
		Name:   snaker.SnakeToCamel(name),
		DBName: name,
	}
	t.Columns = ins.inspectColumns(name)
	t.Indexes = ins.inspectIndex(name)
	for _, idx := range t.Indexes {
		if idx.Primary {
			t.PrimaryKey = idx
		} else {
			t.NonPrimaryIndexes = append(t.NonPrimaryIndexes, idx)
		}
	}
	for _, clm := range t.Columns {
		t.ColumnDBNames = append(t.ColumnDBNames, clm.DBName)
		if t.SelectFields != "" {
			t.SelectFields += ","
			t.ScanFields += ","
		}
		t.SelectFields += "`" + clm.DBName + "`"
		t.ScanFields += "&r." + clm.Name
	}
	return t
}

var typeRe = regexp.MustCompile(`([a-z]+)\(?(\d+\,?\d*)?\)?\s*(unsigned)?`)

var typeMap = map[string]string{
	"tinyint":   "int8",
	"smallint":  "int16",
	"mediumint": "int32",
	"int":       "int32",
	"bigint":    "int64",
	"decimal":   "float64",
	"float":     "float32",
	"double":    "float64",
	"char":      "string",
	"varchar":   "string",
	"text":      "string",
	"blob":      "[]byte",
	"date":      "time.Time",
	"datetime":  "time.Time",
	"timestamp": "time.Time",
}

func (ins *Inspector) inspectColumns(table string) []Column {
	columns := []Column{}
	rows, err := ins.db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", table))
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var fName, fType, fNull, fKey, fExtra string
		var fDef sql.NullString
		if err := rows.Scan(&fName, &fType, &fNull, &fKey, &fDef, &fExtra); err != nil {
			log.Panic(err)
		}
		c := Column{
			Name:     snaker.SnakeToCamel(fName),
			DBName:   fName,
			DBType:   fType,
			Nullable: fNull == "YES",
			Extra:    fExtra,
		}
		if fDef.Valid {
			c.Default = fDef.String
		} else {
			c.Default = "NULL"
		}
		t := typeRe.FindSubmatch([]byte(fType))
		c.Type = typeMap[string(t[1])]
		if c.Type == "" {
			log.Panicf("Undefined type %s", fType)
		}
		if 0 < len(t[2]) {
			sz := string(t[2])
			if c.Type == "float64" {
				s := strings.Split(sz, ",")
				sz = s[0]
			}
			size, err := strconv.ParseUint(sz, 10, 32)
			if err != nil {
				log.Panic(err)
			}
			c.Size = uint32(size)
		}
		if 0 < len(t[3]) {
			c.Type = "u" + c.Type
		}
		if c.Nullable && c.Type == "string" {
			c.Type = "sql.NullString"
		}
		columns = append(columns, c)
	}
	return columns
}

func (ins *Inspector) inspectIndex(table string) []Index {
	indexes := []Index{}
	rows, err := ins.db.Query(fmt.Sprintf("SHOW INDEX FROM %s", table))
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	idx := 0
	for rows.Next() {
		var iTable, iName, iColName, iNull, iIndexType, iComment, iIndexComment string
		var iNonUnique, iSeqIdx, iCardinality uint32
		var iCollation, iSubPart, iPacked sql.NullString
		if err := rows.Scan(&iTable, &iNonUnique, &iName, &iSeqIdx, &iColName, &iCollation, &iCardinality, &iSubPart, &iPacked, &iNull, &iIndexType, &iComment, &iIndexComment); err != nil {
			log.Panic(err)
		}
		if iSeqIdx == 1 {
			indexes = append(indexes, Index{
				Name:    iName,
				Unique:  iNonUnique == 0,
				Primary: iName == "PRIMARY",
			})
			idx = len(indexes) - 1
		}
		if indexes[idx].Name != iName {
			log.Panic("Index is not seqencial")
		}
		indexes[idx].Columns = append(indexes[idx].Columns, iColName)
	}
	return indexes
}
