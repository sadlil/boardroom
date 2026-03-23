package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/internal/llm"
	"github.com/sadlil/boardroom/internal/llm/fake"
	"github.com/sadlil/boardroom/internal/llm/gemini"
	"github.com/sadlil/boardroom/internal/llm/ollama"
	"github.com/sadlil/boardroom/internal/server"
)

var envFile = flag.String("env", "", "Path to the .env file to load")

func main() {
	flag.Parse()

	// Dynamically load the .env file values into runtime variables only if specified
	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			log.Printf("Warning: Failed to load .env file at %s: %v", *envFile, err)
		} else {
			log.Printf("Loaded environment variables from %s", *envFile)
		}
	} else {
		log.Println("Note: No -env flag provided; executing strictly with system environment variables.")
	}

	log.Println("Initializing Boardroom...")
	log.Printf("LLM Provider: %s", os.Getenv("LLM_PROVIDER"))
	log.Printf("LLM Model: %s", os.Getenv("LLM_MODEL"))

	// Initialize LLM Client via dependency injection
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}

	var llmClient llm.Client
	switch provider {
	case "ollama":
		client, err := ollama.NewClient()
		if err != nil {
			log.Fatalf("Failed to initialize Ollama client: %v", err)
		}
		llmClient = client
	case "gemini":
		client, err := gemini.NewClient(os.Getenv("GEMINI_API_KEY"), os.Getenv("LLM_MODEL"))
		if err != nil {
			log.Fatalf("Failed to initialize Gemini client: %v", err)
		}
		llmClient = client
	case "fake", "mock":
		llmClient = fake.NewClient()
	default:
		log.Fatalf("Unsupported LLM provider: %s", provider)
	}

	storageRoot := os.Getenv("STORAGE_ROOT")
	if storageRoot == "" {
		storageRoot = "./data"
	}

	sqlPath := filepath.Join(storageRoot, "sql", "boardroom.db")
	vectorPath := filepath.Join(storageRoot, "vector")

	// Initialize DBs
	sqlite, err := database.NewSQLiteDB(sqlPath)
	if err != nil {
		log.Fatalf("Failed to init SQLite: %v", err)
	}
	defer sqlite.Close()

	memory, err := database.NewVectorMemory(vectorPath)
	if err != nil {
		log.Fatalf("Failed to init Vector Memory: %v", err)
	}

	// Initialize Orchestrator
	orchestrator := agents.NewOrchestrator(llmClient, sqlite, memory)

	// Setup Server
	srv := server.NewServer(sqlite, memory, orchestrator)

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("Starting Server on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down the server gracefully...")
	if err := srv.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}
