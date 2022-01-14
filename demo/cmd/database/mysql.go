package database

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Refer to https://github.com/openservicemesh/osm-docs/blob/main/manifests/apps/mysql.yaml for database setup
const (
	dbuser = "root"
	dbpass = "mypassword"
	dbport = 3306
	dbname = "booksdemo"
)

// GetMySQLConnection returns a MySQL connection using default configuration
func GetMySQLConnection() (*gorm.DB, error) {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&timeout=20s", dbuser, dbpass, "mysql.bookwarehouse", dbport, dbname)
	db, err := gorm.Open(mysql.Open(connStr), &gorm.Config{})

	return db, err
}
