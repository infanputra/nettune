# Contributing to Nettune

Thank you for your interest in contributing to Nettune! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment. Please:

- Be respectful and constructive in discussions
- Welcome newcomers and help them get started
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

### Prerequisites

- **Go**: Version 1.24.2 or later
- **Bun** or **Node.js**: For the JS wrapper package
- **Make**: For build automation
- **Git**: For version control

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/nettune.git
cd nettune
```

3. Add the upstream remote:

```bash
git remote add upstream https://github.com/jtsang4/nettune.git
```

## Development Setup

### Go Backend

```bash
# Install dependencies
go mod download

# Build the binary
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### JavaScript Wrapper

```bash
cd js

# Install dependencies
bun install

# Build
bun run build

# Run tests
bun test

# Type check
bun run typecheck
```

### Running Locally

```bash
# Start the server (requires root for system modifications)
sudo make run-server

# In another terminal, start the client
make run-client
```

## How to Contribute

### Reporting Bugs

Before submitting a bug report:

1. Check the [existing issues](https://github.com/jtsang4/nettune/issues) to avoid duplicates
2. Collect information about the bug:
   - OS and version
   - Go version (`go version`)
   - Nettune version (`nettune version`)
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs

Create a new issue using the bug report template.

### Suggesting Features

We welcome feature suggestions! Please:

1. Check existing issues and discussions for similar ideas
2. Clearly describe the use case and benefits
3. Consider implementation complexity
4. Be open to discussion and feedback

### Contributing Code

1. **Find an issue** to work on, or create one for discussion
2. **Comment on the issue** to express your interest
3. **Wait for assignment** or approval before starting work
4. **Create a branch** for your changes
5. **Submit a pull request** when ready

## Pull Request Process

### Branch Naming

Use descriptive branch names:

- `feature/add-cake-profile` - New features
- `fix/rtt-calculation-error` - Bug fixes
- `docs/update-readme` - Documentation updates
- `refactor/simplify-adapter` - Code refactoring
- `test/add-api-tests` - Test additions

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, semicolons, etc)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:

```
feat(profiles): add low-latency optimization profile

fix(server): correct RTT calculation for high-latency links

docs(readme): update installation instructions
```

### PR Checklist

Before submitting your PR:

- [ ] Code follows the project's coding standards
- [ ] All tests pass (`make test`)
- [ ] New code has appropriate test coverage
- [ ] Documentation is updated if needed
- [ ] Commit messages follow conventions
- [ ] PR description clearly explains the changes

### Review Process

1. Maintainers will review your PR
2. Address any requested changes
3. Once approved, a maintainer will merge the PR

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write descriptive variable and function names
- Add comments for exported functions and types
- Handle errors explicitly

```go
// Good
func (s *Server) GetProfile(id string) (*Profile, error) {
    if id == "" {
        return nil, ErrInvalidProfileID
    }
    // ...
}

// Avoid
func (s *Server) GetProfile(id string) (*Profile, error) {
    // Missing validation
    // ...
}
```

### JavaScript/TypeScript Code

- Use TypeScript for type safety
- Follow existing code patterns
- Run type checking before committing
- Use descriptive names

### Error Handling

- Return errors rather than panicking
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use custom error types for specific error conditions

### Logging

- Use structured logging with zap
- Include relevant context in log messages
- Use appropriate log levels:
  - `Debug`: Detailed information for debugging
  - `Info`: General operational information
  - `Warn`: Warning conditions
  - `Error`: Error conditions

## Testing Guidelines

### Writing Tests

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Test both success and error cases

```go
func TestProfileValidation(t *testing.T) {
    tests := []struct {
        name    string
        profile *Profile
        wantErr bool
    }{
        {
            name:    "valid profile",
            profile: &Profile{ID: "test", Name: "Test"},
            wantErr: false,
        },
        {
            name:    "empty ID",
            profile: &Profile{ID: "", Name: "Test"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateProfile(tt.profile)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateProfile() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Coverage

- Aim for at least 70% coverage for new code
- Focus on critical paths and edge cases
- Don't write tests just to increase coverage numbers

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/server/service/...
```

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use complete sentences in comments
- Include examples for complex functionality

```go
// ApplyProfile applies the specified profile to the system.
// It creates a snapshot before applying changes and supports
// automatic rollback if the changes cause connectivity issues.
//
// Parameters:
//   - profileID: The ID of the profile to apply
//   - dryRun: If true, only show what would change without applying
//   - autoRollbackSeconds: Seconds to wait before auto-rollback (0 = disabled)
//
// Returns an ApplyResult containing the changes made and any verification results.
func (s *ApplyService) ApplyProfile(profileID string, dryRun bool, autoRollbackSeconds int) (*ApplyResult, error)
```

### README Updates

When adding new features, update the README to include:

- Feature description
- Usage examples
- Configuration options
- Any new dependencies

## Security

If you discover a security vulnerability:

1. **Do not** open a public issue
2. Email the maintainers directly
3. Provide detailed information about the vulnerability
4. Allow time for a fix before public disclosure

## Questions?

If you have questions about contributing:

1. Check existing documentation
2. Search closed issues for answers
3. Open a new discussion on GitHub

Thank you for contributing to Nettune! 🎉
