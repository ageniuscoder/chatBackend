package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/ageniuscoder/mmchat/backend/internal/config"
	"github.com/ageniuscoder/mmchat/backend/internal/storage/sqlite"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Entry point of MmChat")
	migrate := flag.Bool("migrate", false, "run migrations and exits")
	flag.Parse()
	//config part
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error Loading Env file: %v", err)
	}
	cfg := config.MustLoad()

	//database handling
	conn, err := sqlite.New(cfg.SQLITEDsn)
	if err != nil {
		log.Fatalf("Error loading to database: %v", err)
	}
	defer conn.Db.Close()

	if *migrate {
		if err := conn.Migrate(); err != nil {
			log.Fatalf("Migration failed %v", err)
		}
		slog.Info("Migration Completed")
		return
	}
}
