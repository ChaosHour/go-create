# Contributing to go-create

Thank you for your interest in contributing to go-create! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Coding Standards](#coding-standards)

## Code of Conduct

This project follows a standard code of conduct. Please be respectful and constructive in all interactions.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Add the upstream repository as a remote
4. Create a feature branch for your changes

```bash
git clone https://github.com/YOUR_USERNAME/go-create.git
cd go-create
git remote add upstream https://github.com/ChaosHour/go-create.git
git checkout -b feature/your-feature-name
```

## Development Setup

### Requirements

- Go 1.20 or higher
- golangci-lint (for linting)
- MySQL 5.7 or 8.0+ (for integration testing)

### Install Development Tools

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install goimports
go install golang.org/x/tools/cmd/goimports@latest
```

### Build the Project

```bash
make build
```

### Run Tests

```bash
make test
```

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-new-feature` - For new features
- `bugfix/fix-issue-123` - For bug fixes
- `docs/update-readme` - For documentation updates
- `refactor/improve-code` - For refactoring

### Commit Messages

Write clear, concise commit messages:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain what and why, not how.

- Bullet points are okay
- Use present tense ("Add feature" not "Added feature")
- Reference issues: Fixes #123
```

### Code Style

This project follows standard Go conventions:

1. **Formatting**: Use `gofmt` and `goimports`
   ```bash
   gofmt -w .
   goimports -w .
   ```

2. **Linting**: Code must pass golangci-lint
   ```bash
   make lint
   ```

3. **Error Handling**: Always check and handle errors appropriately
   ```go
   // Bad
   data, _ := ioutil.ReadFile("file.txt")
   
   // Good
   data, err := ioutil.ReadFile("file.txt")
   if err != nil {
       return fmt.Errorf("reading file: %w", err)
   }
   ```

4. **Documentation**: Add GoDoc comments for exported functions
   ```go
   // CreateUser creates a new MySQL user with the specified password.
   // It validates the password against the configured policy and returns
   // an error if validation fails.
   func CreateUser(username, password string) error {
       // ...
   }
   ```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests for a specific package
go test -v ./pkg/auth/...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...
```

### Writing Tests

1. **Use table-driven tests** for multiple scenarios:
   ```go
   func TestFunction(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           want    string
           wantErr bool
       }{
           // test cases
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // test logic
           })
       }
   }
   ```

2. **Test both success and error paths**

3. **Use meaningful test names** that describe what is being tested

4. **Keep tests focused** - one test should verify one behavior

### Test Coverage

- Aim for at least 70% coverage for new code
- Core packages (auth, config, database) should have high coverage
- Add tests for bug fixes to prevent regressions

## Submitting Changes

### Before Submitting

1. **Run all tests**: `make test`
2. **Run linter**: `make lint`
3. **Format code**: `gofmt -w . && goimports -w .`
4. **Update documentation** if needed
5. **Rebase on upstream main** to avoid merge conflicts

```bash
git fetch upstream
git rebase upstream/main
```

### Pull Request Process

1. Push your changes to your fork
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a pull request on GitHub with:
   - Clear title describing the change
   - Description of what changed and why
   - Reference to any related issues
   - Screenshots for UI changes (if applicable)

3. Address review feedback promptly

4. Ensure CI checks pass

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe testing performed

## Checklist
- [ ] Tests pass locally
- [ ] Linter passes
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

## Coding Standards

### Security

1. **Never log passwords** in plain text
2. **Sanitize error messages** that might contain sensitive data
3. **Use secure temp files** with proper permissions
4. **Validate all inputs** from users

### Performance

1. **Use connection pooling** for database connections
2. **Close resources** properly (use defer)
3. **Avoid unnecessary allocations** in hot paths

### Error Handling

1. **Use error wrapping** with `fmt.Errorf("context: %w", err)`
2. **Return errors** from library code, don't call `log.Fatal`
3. **Provide context** in error messages

### Documentation

1. **Add GoDoc comments** to all exported functions and types
2. **Update README** when adding features
3. **Add examples** for complex functionality
4. **Keep docs in sync** with code

## Questions?

If you have questions about contributing, please:

1. Check existing issues and pull requests
2. Open a new issue for discussion
3. Ask in the pull request comments

Thank you for contributing to go-create!
