package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/golang/glog"
	"github.com/joho/godotenv"
	"github.com/sadlil/boardroom/internal/agents"
	"github.com/sadlil/boardroom/internal/database"
	"github.com/sadlil/boardroom/internal/llm"
	"github.com/sadlil/boardroom/internal/llm/anthropic"
	"github.com/sadlil/boardroom/internal/llm/deepseek"
	"github.com/sadlil/boardroom/internal/llm/fake"
	"github.com/sadlil/boardroom/internal/llm/gemini"
	"github.com/sadlil/boardroom/internal/llm/groq"
	"github.com/sadlil/boardroom/internal/llm/mistral"
	"github.com/sadlil/boardroom/internal/llm/ollama"
	"github.com/sadlil/boardroom/internal/llm/openai"
	"github.com/sadlil/boardroom/internal/llm/openrouter"
	"github.com/sadlil/boardroom/internal/llm/xai"
	"github.com/sadlil/boardroom/internal/server"
)

var envFile = flag.String("env", "", "Path to the .env file to load")

func main() {
	// Force glog to log to stderr natively instead of to temp files
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Dynamically load the .env file values into runtime variables only if specified
	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			glog.Warningf("Failed to load .env file path=%s error=%v", *envFile, err)
		} else {
			glog.Infof("Loaded environment variables path=%s", *envFile)
		}
	} else {
		glog.Info("No -env flag provided; executing strictly with system environment variables.")
	}

	glog.Info("Initializing Boardroom...")

	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}
	model := os.Getenv("LLM_MODEL")

	glog.Infof("LLM Configuration provider=%s model=%s", provider, model)

	var llmClient llm.Client
	switch provider {
	case "ollama":
		client, err := ollama.NewClient()
		if err != nil {
			glog.Errorf("Failed to initialize Ollama client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "gemini":
		client, err := gemini.NewClient(os.Getenv("GEMINI_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize Gemini client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "openai":
		client, err := openai.NewClient(os.Getenv("OPENAI_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize OpenAI client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "anthropic":
		client, err := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize Anthropic client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "xai":
		client, err := xai.NewClient(os.Getenv("XAI_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize xAI client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "groq":
		client, err := groq.NewClient(os.Getenv("GROQ_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize Groq client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "openrouter":
		client, err := openrouter.NewClient(os.Getenv("OPENROUTER_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize OpenRouter client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "deepseek":
		client, err := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize DeepSeek client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "mistral":
		client, err := mistral.NewClient(os.Getenv("MISTRAL_API_KEY"), model)
		if err != nil {
			glog.Errorf("Failed to initialize Mistral client error=%v", err)
			os.Exit(1)
		}
		llmClient = client
	case "fake", "mock":
		llmClient = fake.NewClient()
	default:
		glog.Errorf("Unsupported LLM provider provider=%s", provider)
		os.Exit(1)
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
		glog.Errorf("Failed to init SQLite error=%v", err)
		os.Exit(1)
	}
	defer sqlite.Close()

	// Always use the local hash-based embedding — no external API calls needed
	glog.Info("Using local FNV hash-based embedding (fully offline, no API key required)")
	memory, err := database.NewVectorMemory(vectorPath, database.LocalEmbeddingFunc())
	if err != nil {
		glog.Errorf("Failed to init Vector Memory error=%v", err)
		os.Exit(1)
	}

	// Initialize Orchestrator
	orchestrator := agents.NewOrchestrator(llmClient, sqlite, memory)

	// Setup Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := server.NewServer(sqlite, memory, orchestrator, port)

	go func() {
		glog.Infof("Starting Server port=%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Errorf("Server startup failed error=%v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	glog.Info("Shutting down the server gracefully...")
	if err := srv.Shutdown(); err != nil {
		glog.Errorf("Server forced to shutdown error=%v", err)
		os.Exit(1)
	}
	glog.Info("Server exiting")
}
