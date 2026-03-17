# CRUSH.md - Fantasy AI SDK

## Build/Test/Lint Commands
- **Build**: `go build ./...`
- **Test all**: `task test` or `go test ./... -count=1`
- **Test single**: `go test -run TestName ./package -v`
- **Test with args**: `task test -- -v -run TestName`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `gofmt -s -w .`
- **Modernize**: `task modernize` or `modernize -fix ./...`

## Code Style Guidelines
- **Package naming**: lowercase, single word (ai, openai, anthropic, google)
- **Imports**: standard library first, then third-party, then local packages
- **Error handling**: Use custom error types with structured fields, wrap with context
- **Types**: Use type aliases for function signatures (`type Option = func(*options)`)
- **Naming**: CamelCase for exported, camelCase for unexported
- **Constants**: Use const blocks with descriptive names (ProviderName, DefaultURL)
- **Structs**: Embed anonymous structs for composition (APICallError embeds *AIError)
- **Functions**: Return error as last parameter, use context.Context as first param
- **Testing**: Use testify/assert, table-driven tests, recorder pattern for HTTP mocking
- **Comments**: Godoc format for exported functions, explain complex logic inline
- **JSON**: Use struct tags for marshaling, handle empty values gracefully

## Project Structure
- `/ai` - Core AI abstractions and agent logic
- `/openai`, `/anthropic`, `/google` - Provider implementations
- `/providertests` - Cross-provider integration tests with VCR recordings
- `/examples` - Usage examples for different patterns