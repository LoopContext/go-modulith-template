# Contributing to Go Modulith Template

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## 🚀 Quick Start

1. **Fork the repository**
2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/go-modulith-template.git
   cd go-modulith-template
   ```

3. **Install dependencies**:
   ```bash
   just install-deps
   ```

4. **Start infrastructure**:
   ```bash
   just docker-up
   ```

5. **Run tests**:
   ```bash
   just test
   just lint
   ```

## 📋 Contribution Process

### 1. Create a branch for your feature

```bash
git checkout -b feature/descriptive-name
```

### 2. Make your changes

Ensure you follow the project conventions:

- **Go Code**: Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- **Commits**: Use descriptive commit messages
- **Tests**: Add tests for new functionality
- **Documentation**: Update relevant documentation

### 3. Run validations

**MANDATORY before committing:**

```bash
# Linter (must pass with 0 errors)
just lint

# Tests
just test

# Coverage (optional but recommended)
just coverage-report
```

### 4. Commit and Push

```bash
git add .
git commit -m "feat: brief description of change"
git push origin feature/descriptive-name
```

### 5. Create a Pull Request

- Use a descriptive title
- Explain what changes and why
- Reference related issues (if applicable)
- Ensure CI passes

## 🔍 Style Guidelines

### Go Code

- **Linting**: The project uses `golangci-lint` with strict configuration
- **Formatting**: All code must pass `gofmt` and `goimports`
- **Naming**: Follow standard Go conventions
- **Errors**: Always wrap errors with context using `fmt.Errorf("context: %w", err)`

### Tests

- **Unit tests**: For business logic (use `gomock` mocks)
- **Integration tests**: For DB operations (with `-short` flag)
- **Minimum coverage**: Aim for >60% on new code

### Documentation

- **README**: Update if you add visible features
- **Code**: Document public functions/types with GoDoc
- **Docs**: Update documents in `/docs/` if relevant

## 📝 Commit Types

Use semantic prefixes:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Refactoring without behavior change
- `test:` - Add or modify tests
- `chore:` - Build, deps, etc. changes

### Updating CHANGELOG.md

When adding features, fixes, or important changes, update `CHANGELOG.md`:

1. Add your change in the appropriate `[Unreleased]` section
2. Use categories: Added, Changed, Deprecated, Removed, Fixed, Security
3. Follow the existing format
4. Changes will be moved to a specific version in the next release

## 🐛 Reporting Bugs

When reporting a bug, include:

1. **Description**: What you expected vs what happened
2. **Steps to reproduce**
3. **Go version**: `go version`
4. **Relevant logs**: If applicable

## 💡 Prloopcontexting Features

To prloopcontexte new functionality:

1. **Open an issue** first to discuss it
2. Explain the use case
3. Consider the architectural impact
4. Wait for feedback before implementing

## ⚠️ Important Considerations

### Do Not Modify Without Justification

- `.golangci.yaml` - Do not relax linting rules
- `sqlc.yaml` - Only change for new modules
- `buf.yaml` - Standard protobuf configuration

### Architecture

This is a **modulith** template:
- Maintain isolation between modules
- Use events for cross-module communication
- Follow the registry pattern for DI
- Document important architectural decisions

### Performance

- Don't optimize prematurely
- If you add performance-critical code, include benchmarks
- Use `go test -bench=.` to validate

## 🤝 Code of Conduct

- Be respectful and constructive
- Accept feedback with an open mind
- Focus on the code, not the people
- Help other contributors when you can

## 📧 Contact

If you have questions, open an issue or discussion on GitHub.

---

Thank you for contributing! 🚀
