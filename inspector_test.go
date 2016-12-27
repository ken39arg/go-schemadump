package schemadump

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	mysqltest "github.com/lestrrat/go-test-mysqld"
)

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
			DBName: "item",
			Columns: []Column{
				Column{"ID", "uint64", false, 20, "NULL", true, "auto_increment", "id", "bigint(20) unsigned"},
				Column{"Name", "string", false, 100, "NULL", false, "", "name", "varchar(100)"},
				Column{"Description", "sql.NullString", true, 0, "NULL", false, "", "description", "text"},
				Column{"Valid", "int8", false, 1, "0", false, "", "valid", "tinyint(1)"},
				Column{"ScoreRate", "float64", false, 10, "1.00", false, "", "score_rate", "decimal(10,2)"},
				Column{"CreatedAt", "time.Time", false, 0, "NULL", false, "", "created_at", "datetime"},
			},
			Indexes: []Index{
				Index{"PRIMARY", []string{"id"}, true, true},
				Index{"name_uniq", []string{"name"}, true, false},
				Index{"valid_created_at_idx", []string{"valid", "created_at"}, false, false},
			},
			PrimaryKey: Index{"PRIMARY", []string{"id"}, true, true},
			NonPrimaryIndexes: []Index{
				Index{"name_uniq", []string{"name"}, true, false},
				Index{"valid_created_at_idx", []string{"valid", "created_at"}, false, false},
			},
			ColumnDBNames: []string{"id", "name", "description", "valid", "score_rate", "created_at"},
			SelectFields:  "`id`,`name`,`description`,`valid`,`score_rate`,`created_at`",
			ScanFields:    "&r.ID,&r.Name,&r.Description,&r.Valid,&r.ScoreRate,&r.CreatedAt",
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
			DBName: "user",
			Columns: []Column{
				Column{"ID", "uint64", false, 20, "NULL", true, "auto_increment", "id", "bigint(20) unsigned"},
				Column{"Name", "string", false, 100, "NULL", false, "", "name", "varchar(100)"},
				Column{"UID", "string", false, 10, "NULL", false, "", "uid", "char(10)"},
				Column{"CreatedAt", "time.Time", false, 0, "NULL", false, "", "created_at", "datetime"},
			},
			Indexes: []Index{
				Index{"PRIMARY", []string{"id"}, true, true},
				Index{"uid", []string{"uid"}, true, false},
				Index{"name", []string{"name"}, false, false},
			},
			PrimaryKey: Index{"PRIMARY", []string{"id"}, true, true},
			NonPrimaryIndexes: []Index{
				Index{"uid", []string{"uid"}, true, false},
				Index{"name", []string{"name"}, false, false},
			},
			ColumnDBNames: []string{"id", "name", "uid", "created_at"},
			SelectFields:  "`id`,`name`,`uid`,`created_at`",
			ScanFields:    "&r.ID,&r.Name,&r.UID,&r.CreatedAt",
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
			DBName: "user_item",
			Columns: []Column{
				Column{"UserID", "uint64", false, 20, "NULL", false, "", "user_id", "bigint(20) unsigned"},
				Column{"ItemID", "uint64", false, 20, "NULL", false, "", "item_id", "bigint(20) unsigned"},
				Column{"Sort", "float32", false, 0, "0", false, "", "sort", "float"},
				Column{"Power", "float64", false, 0, "0", false, "", "power", "double"},
				Column{"CreatedAt", "time.Time", false, 0, "NULL", false, "", "created_at", "datetime"},
			},
			Indexes: []Index{
				Index{"PRIMARY", []string{"user_id", "item_id"}, true, true},
				Index{"item_id", []string{"item_id"}, false, false},
			},
			PrimaryKey: Index{"PRIMARY", []string{"user_id", "item_id"}, true, true},
			NonPrimaryIndexes: []Index{
				Index{"item_id", []string{"item_id"}, false, false},
			},
			ColumnDBNames: []string{"user_id", "item_id", "sort", "power", "created_at"},
			SelectFields:  "`user_id`,`item_id`,`sort`,`power`,`created_at`",
			ScanFields:    "&r.UserID,&r.ItemID,&r.Sort,&r.Power,&r.CreatedAt",
		},
	},

	{
		ddl: `CREATE TABLE no_index (
				foo    varchar(10) not null,
				val    varchar(255),
				created_at datetime not null
			)`,
		expect: Table{
			Name:   "NoIndex",
			DBName: "no_index",
			Columns: []Column{
				Column{"Foo", "string", false, 10, "NULL", false, "", "foo", "varchar(10)"},
				Column{"Val", "sql.NullString", true, 255, "NULL", false, "", "val", "varchar(255)"},
				Column{"CreatedAt", "time.Time", false, 0, "NULL", false, "", "created_at", "datetime"},
			},
			Indexes:           []Index{},
			PrimaryKey:        Index{},
			NonPrimaryIndexes: []Index{},
			ColumnDBNames:     []string{"foo", "val", "created_at"},
			SelectFields:      "`foo`,`val`,`created_at`",
			ScanFields:        "&r.Foo,&r.Val,&r.CreatedAt",
		},
	},
}

var db *sql.DB

func TestMain(m *testing.M) {
	runner := func() int {
		var err error
		var mysqld *mysqltest.TestMysqld
		mysqld, err = mysqltest.NewMysqld(nil)
		if err != nil {
			log.Fatalf("Failed to start mysqld: %s", err)
			return 1
		}
		defer mysqld.Stop()

		db, err = sql.Open("mysql", mysqld.Datasource("test", "", "", 0))
		if err != nil {
			log.Fatalf("Failed to Open mysqld: %s", err)
			return 1
		}
		defer db.Close()

		for _, test := range testCase {
			_, e := db.Exec(test.ddl)
			if e != nil {
				log.Fatalf("ddl failed: %s SQL{ %s }", e, test.ddl)
				return 1
			}
		}
		return m.Run()
	}
	os.Exit(runner())
}

func TestNewInspect(t *testing.T) {
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
		f := fmt.Sprintf("%q", found)
		e := fmt.Sprintf("%q", expect)
		if f != e {
			t.Errorf("%s => \n\tgot  : %s\n\twant : %s", found.Name, f, e)
			continue
		}
	}
}

func TestInspectTables(t *testing.T) {
	inspecter := &Inspector{db: db}
	inspecter.InspectTables("user", "item", "non")
	if len(inspecter.Tables) != 2 {
		t.Errorf("Inspect with tables. table size => %d want 2", len(inspecter.Tables), 2)
		return
	}
}
