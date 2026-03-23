# Deployment & Running Instructions

This guide provides detailed instructions on how to run **The Boardroom** using either pre-built Docker images or by building from source.

## 1. Quick Start (Docker)

The fastest way to get started is using the official Docker image.

```bash
# Run with environment variables directly
docker run -p 8080:8080 \
  -e LLM_PROVIDER=gemini \
  -e GEMINI_API_KEY=your_key \
  -e LLM_MODEL=gemini-2.5-flash \
  sadlil/boardroom:latest
```

---

## 2. Running from Source

If you have Go installed, you can build and run the binary manually.

### Build
> [!IMPORTANT]
> This project requires **CGO** for SQLite support. Ensure you have a C compiler (`gcc` or `clang`) installed.

```bash
go build -o boardroom ./cmd/boardroom
```

### Run with .env file
The application **only** loads a `.env` file if the `-env` flag is explicitly provided:
```bash
./boardroom -env=.env
```

### Run with system environment variables
If no flag is provided, the application strictly uses your shell environment:
```bash
export LLM_PROVIDER=ollama
./boardroom
```

---

## 3. Provider Configurations

### Ollama (Local)
1. Ensure Ollama is running.
2. `ollama pull gemma3:1b`
3. Configuration:
   - `LLM_PROVIDER=ollama`
   - `LLM_MODEL=gemma3:1b`
   - `OLLAMA_HOST=http://localhost:11434` (default)

### Google Gemini (Cloud)
1. Set `LLM_PROVIDER=gemini`
2. Set `LLM_MODEL=gemini-1.5-flash` (or `gemini-1.5-pro`)
3. Set `GEMINI_API_KEY=your_key`

### Fake LLM (Testing)
- Set `LLM_PROVIDER=fake` to use the built-in mock board for development.

---

## 4. Configuration Reference

| Variable | Description | Default |
|---|---|---|
| `-env` flag | **Required** to load a `.env` file. | (none) |
| `PORT` | Local port for the server. | `8080` |
| `STORAGE_ROOT` | Path for DB and Vector index data. | `./data` |
| `LLM_PROVIDER` | `ollama`, `gemini`, or `fake`. | `ollama` |
| `LLM_MODEL` | The specific model ID to use. | `gemma3:1b` |
| `MAX_SESSIONS` | Max sessions in LRU memory cache. | `100` |
| `MAX_CONCURRENT_AGENTS` | Max parallel LLM inferences. | `3` |
| `GEMINI_API_KEY` | Google AI Studio Key. | (Required for Gemini) |

---

## 5. Docker Volume Persistence
To persist your session history and vector memory when using Docker, mount a volume to `/root/data`:

```bash
docker run -p 8080:8080 \
  -v $(pwd)/data:/root/data \
  -e LLM_PROVIDER=ollama \
  -e LLM_MODEL=gemma3:1b \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  sadlil/boardroom:latest
```
