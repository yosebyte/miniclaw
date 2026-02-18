# 🐾 miniclaw

A minimal personal AI assistant in Go, powered by Claude, delivered via Telegram.

> Inspired by [nanobot](https://github.com/HKUDS/nanobot) — ~99% smaller, Go-native, single binary.

## Features

- **Claude OAuth** — login with your Claude.ai account (no API key needed); API key also supported
- **Telegram only** — long polling, typing indicator, Markdown→HTML, allow-list
- **Agent loop** — tool-calling, up to 20 iterations per message
- **Built-in tools** — `read_file`, `write_file`, `edit_file`, `list_dir`, `exec`, `web_fetch`
- **Memory** — long-term `MEMORY.md` + `HISTORY.md`, auto-consolidated from session history
- **Single binary** — `go build -o miniclaw .`

## Quick Start

```bash
# 1. Build
git clone https://github.com/yosebyte/miniclaw
cd miniclaw
go build -o miniclaw .

# 2. Initialise config
./miniclaw onboard

# 3. Authenticate with Claude (OAuth)
./miniclaw provider login

#    OR add your Anthropic API key to ~/.miniclaw/config.json:
#    "provider": { "apiKey": "sk-ant-..." }

# 4. Add Telegram bot token to ~/.miniclaw/config.json:
#    "telegram": { "token": "YOUR_BOT_TOKEN" }
#    Get a token from @BotFather on Telegram.

# 5. Start
./miniclaw gateway
```

## Configuration

Config lives at `~/.miniclaw/config.json`:

```json
{
  "provider": {
    "accessToken": "",
    "refreshToken": "",
    "apiKey": "",
    "model": "claude-opus-4-5",
    "maxTokens": 8192,
    "maxIterations": 20,
    "memoryWindow": 50
  },
  "telegram": {
    "token": "YOUR_BOT_TOKEN",
    "allowFrom": ["YOUR_TELEGRAM_USER_ID"]
  },
  "workspace": "~/.miniclaw/workspace"
}
```

`allowFrom` — list of Telegram user IDs or usernames. Leave empty to allow everyone.

## CLI Reference

| Command | Description |
|---------|-------------|
| `miniclaw onboard` | Create default config and workspace |
| `miniclaw provider login` | OAuth login with claude.ai |
| `miniclaw gateway` | Start Telegram bot (long polling) |
| `miniclaw agent -m "..."` | Single message via CLI |
| `miniclaw agent` | Interactive CLI chat |
| `miniclaw status` | Show auth and config status |

## Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Greet the bot |
| `/new` | Start a new conversation (clears history) |
| `/help` | Show available commands |

## Project Structure

```
miniclaw/
├── main.go
├── cmd/              # CLI commands (cobra)
├── internal/
│   ├── config/       # Config loading/saving
│   ├── provider/     # Claude API + OAuth PKCE flow
│   ├── agent/        # Agent loop, sessions, memory
│   ├── tools/        # Built-in tools (fs, shell, web)
│   └── telegram/     # Telegram bot
```

## License

MIT — Copyright (c) 2026 yosebyte
