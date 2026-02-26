# how

A smart terminal cheatsheet — ask a natural language question, get back a shell command.

```
$ how find all go files modified in the last 24 hours

  $ find . -name '*.go' -mtime -1
  Finds all .go files modified within the last 24 hours
```

## Features

- Natural language to shell command translation
- Multiple LLM backends: **Anthropic**, **OpenAI**, and **Ollama** (local)
- Clean, colorized terminal output
- Quiet mode for piping (`-q`)
- Optional auto-execution (`-y`)

## Installation

### Homebrew

```sh
brew install swibrow/tap/how
```

### From source

```sh
go install github.com/swibrow/how/cmd/how@latest
```

### From releases

Download a prebuilt binary from [Releases](https://github.com/swibrow/how/releases).

## Usage

```sh
# Ask a question
how reverse a string in bash

# Run the suggested command immediately
how -y list listening ports

# Output only the command (useful for piping)
how -q convert png to jpg with imagemagick | sh
```

## Configuration

Initialize a config file:

```sh
how config init
```

This creates `~/.config/how/config.yaml`:

```yaml
provider: anthropic
anthropic:
  api_key: ""
  model: claude-sonnet-4-6
openai:
  api_key: ""
  model: gpt-4o
ollama:
  model: llama3
  url: http://localhost:11434/v1
```

### API keys

Set via environment variables (recommended) or in the config file:

```sh
export ANTHROPIC_API_KEY=sk-...
# or
export OPENAI_API_KEY=sk-...
```

For **Ollama**, no API key is needed — just have Ollama running locally.

### View current config

```sh
how config show
```

## Development

### Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/) (for linting)

### Build & test

```sh
make          # lint + test + build
make test     # run tests with race detector
make lint     # run linters
make coverage # generate coverage report
make build    # compile binary
```

## License

MIT
