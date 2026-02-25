package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/alltomatos/clawproject/internal/api"
	"github.com/alltomatos/clawproject/internal/core"
	"github.com/alltomatos/clawproject/internal/db"
)

//go:embed all:ui
var uiAssets embed.FS

func main() {
	fmt.Println("🦞 ClawProject - Gerenciador de Projetos Agent-Native")

	// 1. Carregar Config do OpenClaw
	cfg, err := core.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar openclaw.json: %v", err)
	}
	fmt.Printf("Config carregada! Gateway na porta: %d\n", cfg.Gateway.Port)

	// 2. Inicializar Banco de Dados (SQLite)
	store, err := db.NewStore()
	if err != nil {
		log.Fatalf("Erro ao inicializar SQLite: %v", err)
	}
	fmt.Println("Banco de dados SQLite pronto!")
	defer store.DB.Close()

	// 3. Inicializar APIs
	apiServer := api.NewServer(store)
	apiServer.RegisterHandlers()

	// 4. Servir Frontend Embutido
	distFS, _ := fs.Sub(uiAssets, "ui")
	
	http.Handle("/", http.FileServer(http.FS(distFS)))

	serverPort := 19192
	fmt.Printf("Dashboard disponível em: http://0.0.0.0:%d\n", serverPort)
	
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", serverPort), nil); err != nil {
		log.Fatal(err)
	}
}
