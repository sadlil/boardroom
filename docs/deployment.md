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

### OpenAI (Cloud)
1. Set `LLM_PROVIDER=openai`
2. Set `LLM_MODEL=gpt-4o` (or `gpt-4-turbo`)
3. Set `OPENAI_API_KEY=your_key`

### Anthropic Claude (Cloud)
1. Set `LLM_PROVIDER=anthropic`
2. Set `LLM_MODEL=claude-3-5-sonnet-latest`
3. Set `ANTHROPIC_API_KEY=your_key`

### xAI Grok (Cloud)
1. Set `LLM_PROVIDER=xai`
2. Set `LLM_MODEL=grok-beta`
3. Set `XAI_API_KEY=your_key`

### Groq (Cloud)
1. Set `LLM_PROVIDER=groq`
2. Set `LLM_MODEL=llama-3.3-70b-versatile`
3. Set `GROQ_API_KEY=your_key`

### OpenRouter (Cloud)
1. Set `LLM_PROVIDER=openrouter`
2. Set `LLM_MODEL=anthropic/claude-3.5-sonnet`
3. Set `OPENROUTER_API_KEY=your_key`

### DeepSeek (Cloud)
1. Set `LLM_PROVIDER=deepseek`
2. Set `LLM_MODEL=deepseek-chat`
3. Set `DEEPSEEK_API_KEY=your_key`

### Mistral (Cloud)
1. Set `LLM_PROVIDER=mistral`
2. Set `LLM_MODEL=mistral-large-latest`
3. Set `MISTRAL_API_KEY=your_key`

### Fake LLM (Testing)
- Set `LLM_PROVIDER=fake` to use the built-in mock board for development.

---

## 4. Configuration Reference

| Variable | Description | Default |
|---|---|---|
| `-env` flag | **Required** to load a `.env` file. | (none) |
| `PORT` | Local port for the server. | `8080` |
| `STORAGE_ROOT` | Path for DB and Vector index data. | `./data` |
| `LLM_PROVIDER` | `ollama`, `gemini`, `openai`, `anthropic`, `xai`, `groq`, `openrouter`, `deepseek`, `mistral`, or `fake`. | `ollama` |
| `LLM_MODEL` | The specific model ID to use. | `gemma3:1b` |
| `MAX_SESSIONS` | Max sessions in LRU memory cache. | `100` |
| `MAX_CONCURRENT_AGENTS` | Max parallel LLM inferences. | `3` |
| `GEMINI_API_KEY` | Google AI Studio Key. | (Required for Gemini) |
| `OPENAI_API_KEY` | OpenAI API Key. | (Required for OpenAI) |
| `ANTHROPIC_API_KEY` | Anthropic API Key. | (Required for Anthropic) |
| `XAI_API_KEY` | xAI API Key. | (Required for xAI) |
| `GROQ_API_KEY` | Groq API Key. | (Required for Groq) |
| `OPENROUTER_API_KEY` | OpenRouter API Key. | (Required for OpenRouter) |
| `DEEPSEEK_API_KEY` | DeepSeek API Key. | (Required for DeepSeek) |
| `MISTRAL_API_KEY` | Mistral API Key. | (Required for Mistral) |

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
