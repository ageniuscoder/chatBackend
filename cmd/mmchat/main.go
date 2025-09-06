package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/config"
	"github.com/ageniuscoder/mmchat/backend/internal/storage/sqlite"
	"github.com/ageniuscoder/mmchat/backend/internal/users"
	"github.com/gin-gonic/gin"
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
	//migration for creating table in database
	if *migrate {
		if err := conn.Migrate(); err != nil {
			log.Fatalf("Migration failed %v", err)
		}
		slog.Info("Migration Completed")
		return
	}
	//http server connection
	r := gin.Default()

	api := r.Group("/api")

	users.RegisterPublic(api, conn.Db, cfg)

	srv := &http.Server{Addr: cfg.Addr, Handler: r}
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("server stopped")
}
