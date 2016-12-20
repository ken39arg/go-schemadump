package schemadump

import (
	"database/sql"
	"log"
	"strings"
	"testing"

	mysqltest "github.com/lestrrat/go-test-mysqld"
)

func CompareColumn(a Column, b Column) bool {
	miss := 0
	if a.Name != b.Name {
		miss++
		log.Println("Name is %q want %q", a.Name, b.Name)
	}
	if a.Type != b.Type {
		miss++
		log.Println("Type is %q want %q", a.Type, b.Type)
	}
	if a.Nullable != b.Nullable {
		miss++
		log.Println("Nullable is %q want %q", a.Nullable, b.Nullable)
	}
	if a.Size != b.Size {
		miss++
		log.Println("Size is %q want %q", a.Size, b.Size)
	}
	if a.Default != b.Default {
		miss++
		log.Println("Default is %q want %q", a.Default, b.Default)
	}
	if a.Extra != b.Extra {
		miss++
		log.Println("Extra is %q want %q", a.Extra, b.Extra)
	}
	if a.dbName != b.dbName {
		miss++
		log.Println("dbName is %q want %q", a.dbName, b.dbName)
	}
	if a.dbType != b.dbType {
		miss++
		log.Println("dbType is %q want %q", a.dbType, b.dbType)
	}
	return miss == 0
}

func CompareColumns(a []Column, b []Column) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !CompareColumn(a[i], b[i]) {
			return false
		}
	}
	return true
}

func CompareIndex(a Index, b Index) bool {
	miss := 0
	if a.Name != b.Name {
		miss++
		log.Println("Name is %q want %q", a.Name, b.Name)
	}
	if strings.Join(a.Columns, ".") != strings.Join(b.Columns, ".") {
		miss++
		log.Println("Columns is %q want %q", a.Columns, b.Columns)
	}
	if a.Unique != b.Unique {
		miss++
		log.Println("Unique is %q want %q", a.Unique, b.Unique)
	}
	if a.Primary != b.Primary {
		miss++
		log.Println("Primary is %q want %q", a.Primary, b.Primary)
	}
	return miss == 0
}

func CompareIndexes(a []Index, b []Index) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !CompareIndex(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestTableFunc(t *testing.T) {
	var testCase = []struct {
		Table  Table
		pk     Index
		others []Index
	}{
		{
			Table: Table{
				Name: "HasPrimary",
				Indexes: []Index{
					Index{"PRIMARY", []string{"id"}, true, true},
					Index{"name_age", []string{"name", "age"}, true, false},
					Index{"foo_var", []string{"foo", "var"}, false, false},
				},
			},
			pk: Index{"PRIMARY", []string{"id"}, true, true},
			others: []Index{
				Index{"name_age", []string{"name", "age"}, true, false},
				Index{"foo_var", []string{"foo", "var"}, false, false},
			},
		},
		{
			Table: Table{
				Name: "NoPrimary",
				Indexes: []Index{
					Index{"name_age", []string{"name", "age"}, true, false},
					Index{"foo_var", []string{"foo", "var"}, false, false},
				},
			},
			pk: Index{"", []string{}, false, false},
			others: []Index{
				Index{"name_age", []string{"name", "age"}, true, false},
				Index{"foo_var", []string{"foo", "var"}, false, false},
			},
		},
	}

	for _, test := range testCase {
		pk := test.Table.PrimaryKey()
		indexes := test.Table.NonPrimaryIndexes()
		if !CompareIndex(pk, test.pk) {
			t.Errorf("%s PrimaryKey => %q want %q", test.Table.Name, pk, test.pk)
		}
		if !CompareIndexes(indexes, test.others) {
			t.Errorf("%s NonPrimaryIndexes => %q want %q", test.Table.Name, indexes, test.others)
		}
	}
}

func TestInspect(t *testing.T) {
	mysqld, err := mysqltest.NewMysqld(nil)
	if err != nil {
		log.Fatalf("Failed to start mysqld: %s", err)
	}
	defer mysqld.Stop()

	db, err := sql.Open("mysql", mysqld.Datasource("test", "", "", 0))
	if err != nil {
		log.Fatalf("Failed to Open mysqld: %s", err)
	}

	var testCase = []struct {
		ddl    string
		expect Table
	}{
		{
			ddl: `CREATE TABLE item (
				id   bigint  unsigned not null auto_increment,
				name varchar(100) not null,
				description text,
				valid tinyint(1) not null default 0,
				score_rate decimal(10, 2) not null default 1.0,
				created_at datetime not null,

				PRIMARY KEY(id),
				UNIQUE name_uniq (name),
				INDEX valid_created_at_idx (valid, created_at)
			)`,
			expect: Table{
				Name:   "Item",
				dbName: "item",
				Columns: []Column{
					Column{"ID", "uint64", false, 20, "NULL", "auto_increment", "id", "bigint(20) unsigned"},
					Column{"Name", "string", false, 100, "NULL", "", "name", "varchar(100)"},
					Column{"Description", "sql.NullString", true, 0, "NULL", "", "description", "text"},
					Column{"Valid", "int8", false, 1, "0", "", "valid", "tinyint(1)"},
					Column{"ScoreRate", "float64", false, 10, "1.00", "", "score_rate", "decimal(10,2)"},
					Column{"CreatedAt", "time.Time", false, 0, "NULL", "", "created_at", "datetime"},
				},
				Indexes: []Index{
					Index{"PRIMARY", []string{"id"}, true, true},
					Index{"name_uniq", []string{"name"}, true, false},
					Index{"valid_created_at_idx", []string{"valid", "created_at"}, false, false},
				},
			},
		},
		{
			ddl: "CREATE TABLE `user` (" +
				"`id`   bigint  unsigned not null auto_increment," +
				"`name` varchar(100) not null," +
				"`uid` char(10) not null," +
				"`created_at` datetime not null," +
				"PRIMARY KEY(`id`)," +
				"UNIQUE `uid` (`uid`)," +
				"INDEX `name` (`name`)" +
				")",
			expect: Table{
				Name:   "User",
				dbName: "user",
				Columns: []Column{
					Column{"ID", "uint64", false, 20, "NULL", "auto_increment", "id", "bigint(20) unsigned"},
					Column{"Name", "string", false, 100, "NULL", "", "name", "varchar(100)"},
					Column{"UID", "string", false, 10, "NULL", "", "uid", "char(10)"},
					Column{"CreatedAt", "time.Time", false, 0, "NULL", "", "created_at", "datetime"},
				},
				Indexes: []Index{
					Index{"PRIMARY", []string{"id"}, true, true},
					Index{"uid", []string{"uid"}, true, false},
					Index{"name", []string{"name"}, false, false},
				},
			},
		},
		{
			ddl: `CREATE TABLE user_item (
				user_id   bigint  unsigned not null,
				item_id   bigint  unsigned not null,
				sort      float   not null default 0,
				power     double  not null default 0,
				created_at datetime not null,

				PRIMARY KEY(user_id, item_id),
				INDEX item_id (item_id)
			)`,
			expect: Table{
				Name:   "UserItem",
				dbName: "user_item",
				Columns: []Column{
					Column{"UserID", "uint64", false, 20, "NULL", "", "user_id", "bigint(20) unsigned"},
					Column{"ItemID", "uint64", false, 20, "NULL", "", "item_id", "bigint(20) unsigned"},
					Column{"Sort", "float32", false, 0, "0", "", "sort", "float"},
					Column{"Power", "float64", false, 0, "0", "", "power", "double"},
					Column{"CreatedAt", "time.Time", false, 0, "NULL", "", "created_at", "datetime"},
				},
				Indexes: []Index{
					Index{"PRIMARY", []string{"user_id", "item_id"}, true, true},
					Index{"item_id", []string{"item_id"}, false, false},
				},
			},
		},
	}

	for _, test := range testCase {
		_, e := db.Exec(test.ddl)
		if e != nil {
			log.Fatalf("ddl failed: %s SQL{ %s }", e, test.ddl)
		}
	}

	inspecter := NewInspector(db)

	if len(inspecter.Tables) != len(testCase) {
		t.Errorf("table size => %d want %d", len(inspecter.Tables), len(testCase))
		return
	}

	for _, test := range testCase {
		expect := test.expect
		var found Table
		for _, table := range inspecter.Tables {
			if table.Name == expect.Name {
				found = table
				break
			}
		}
		if found.Name == "" {
			t.Errorf("table %s not found", expect.Name)
			continue
		}
		if found.dbName != expect.dbName {
			t.Errorf("%s dbName => %s want %s", expect.Name, found.dbName, expect.dbName)
			continue
		}
		if !CompareColumns(found.Columns, expect.Columns) {
			t.Errorf("%s Columns => %q want %q", expect.Name, found.Columns, expect.Columns)
			continue
		}
		if !CompareIndexes(found.Indexes, expect.Indexes) {
			t.Errorf("%s Indexes => %q want %q", expect.Name, found.Indexes, expect.Indexes)
			continue
		}
	}

}
