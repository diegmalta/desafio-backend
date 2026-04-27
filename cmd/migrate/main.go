package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"desafio-backend/internal/config"
	migrator "desafio-backend/internal/migrate"
	gomigrate "github.com/golang-migrate/migrate/v4"
)

func main() {
	databaseURL := flag.String("database", os.Getenv("DATABASE_URL"), "DSN (default: DATABASE_URL)")
	var (
		doUp   = flag.Bool("up", false, "Aplicar todas as migrações pendentes")
		doDown = flag.Int("down", 0, "Reverter N migrações (>=1)")
		doVer  = flag.Bool("version", false, "Mostrar versão actual e sair")
	)
	flag.Parse()
	ctx := context.Background()
	url := *databaseURL
	if url == "" {
		url = config.Load().DatabaseURL
	}
	if url == "" {
		log.Fatal("defina -database= ou DATABASE_URL")
	}

	if *doVer {
		v, dirty, err := migrator.Version(ctx, url)
		if err != nil {
			if errors.Is(err, gomigrate.ErrNilVersion) {
				fmt.Println("version=0 (nenhuma migração) dirty=false")
				return
			}
			log.Fatalf("version: %v", err)
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return
	}
	if *doUp {
		if err := migrator.Up(ctx, url); err != nil {
			log.Fatalf("up: %v", err)
		}
		return
	}
	if *doDown > 0 {
		if err := migrator.Down(ctx, url, *doDown); err != nil {
			log.Fatalf("down: %v", err)
		}
		return
	}
	flag.Usage()
	log.Fatal("indique -up, -down N ou -version")
}
