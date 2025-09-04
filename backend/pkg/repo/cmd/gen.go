package main

import (
	"SYS_DESIGN_PLAYGROUND/pkg/repo/config"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath:        "../model/query",
		Mode:           gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
		FieldCoverable: true,                                                               // generate pointer type for zero value type
	})

	gormdb, _ := gorm.Open(mysql.Open(config.DSN))
	// reuse your gorm db
	g.UseDB(gormdb)

	// Generate basic type-safe DAO API for generated struct `model.User` following conventions
	g.ApplyBasic(g.GenerateAllTable()...)

	// Generate the code
	g.Execute()
}
