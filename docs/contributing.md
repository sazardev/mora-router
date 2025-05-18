# Contributing to MoraRouter

First off, thank you for considering contributing to MoraRouter! It's people like you who make MoraRouter such a great tool. This document provides guidelines and steps for contributing.

## 😎 The Fun Part: Why Contribute?

Before we get to the formal stuff, let's talk about why contributing to MoraRouter is awesome:

- **Build something people actually use!** MoraRouter aims to be the most elegant and powerful router for Go web applications.
- **Learn from code reviews** by experienced Go developers.
- **Level up your Go skills** through practical, real-world coding.
- **Add an impressive open-source contribution** to your portfolio.
- **Solve interesting problems** in HTTP routing, middleware design, and API development.
- **Join a friendly community** of developers passionate about great web infrastructure.

Plus, your name gets immortalized in our contributors list! 🏆

## 🛣️ Development Roadmap

Want to know where MoraRouter is heading? Here are some areas we're focusing on:

- Performance optimizations for high-load applications
- Enhanced WebSocket support
- GraphQL integration
- More middleware extensions
- Expanded testing utilities
- Code generation improvements
- Documentation and examples

If you're interested in these areas, your contributions would be especially welcome!

## 🚀 Getting Started

### 1. Set Up Your Environment

Fork and clone the repository:

```bash
# Clone your fork
git clone https://github.com/your-username/mora-router.git

# Enter the project directory
cd mora-router

# Add the original repo as "upstream"
git remote add upstream https://github.com/yourusername/mora-router.git
```

### 2. Set Up Development Dependencies

```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
```

### 3. Create a Branch

Create a branch for your feature or bugfix:

```bash
# Create a new branch
git checkout -b feature/my-awesome-feature

# Or for bugfixes
git checkout -b fix/annoying-bug
```

### 4. Make Your Changes

Now you're ready to make changes! Here are some tips:

- Keep your changes focused on a single issue/feature
- Write clean, well-commented code that follows Go conventions
- Add tests for your changes
- Update documentation as needed

### 5. Test Your Changes

Run the tests to ensure your changes don't break existing functionality:

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

### 6. Lint Your Code

We use golangci-lint to ensure code quality:

```bash
# Run the linter
golangci-lint run
```

### 7. Submit Your Pull Request

Push your changes and create a pull request:

```bash
# Push your changes to your fork
git push origin feature/my-awesome-feature

# Then go to GitHub and create a Pull Request
```

## 🏗️ Project Structure

Understanding the project structure will help you contribute effectively:

```
mora-router/
├── router/               # Core router package
│   ├── router.go         # Main router implementation
│   ├── middleware.go     # Built-in middleware
│   ├── render.go         # Response rendering utilities
│   ├── form.go           # Form handling and validation
│   ├── websocket.go      # WebSocket support
│   └── ...
├── examples/             # Example applications
│   ├── simple/           # Basic usage examples
│   ├── resource-demo/    # RESTful resource example
│   └── ...
├── docs/                 # Documentation
├── benchmarks/           # Performance benchmarks
└── tests/                # Integration tests
```

## 📝 Coding Guidelines

### Go Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `goimports` to format your code
- Write idiomatic Go code
- Add comments for exported functions, types, and packages

### Testing

- Aim for high test coverage (>80%)
- Write table-driven tests for functions with multiple cases
- Use subtests for cleaner test organization
- Include benchmarks for performance-critical code

### Commits

- Use descriptive commit messages
- Structure commits logically
- Reference issue numbers in commits and pull requests

### Documentation

- Update documentation for new features
- Add examples for complex functionality
- Ensure godoc comments are clear and complete

## 🔄 Pull Request Process

1. Update the README.md or documentation with details of your changes, if applicable
2. Make sure all tests pass
3. Get your code reviewed by at least one maintainer
4. Once approved, a maintainer will merge your PR

## 🐛 Bug Reports

Found a bug? Please create an issue with the following information:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Go version and environment details
- Code sample if possible

## 💡 Feature Requests

Have an idea for MoraRouter? We'd love to hear it! Create an issue with:

- A clear, descriptive title
- Detailed description of the proposed feature
- Any relevant examples or use cases
- Why this would be valuable to MoraRouter users

## 📊 Code Review Process

All submissions require review. We use GitHub pull requests for this purpose.

Reviews focus on:
- Code correctness and quality
- Test coverage
- Documentation
- Performance implications
- API design and usability

## 🙏 Acknowledgements

Your contributions are highly valued! All contributors will be listed in our README and CONTRIBUTORS file.

## 📣 Communication

- **GitHub Issues**: For bug reports, feature requests, and discussions
- **Pull Requests**: For code contributions
- **Discord**: Join our community chat (link in README)

## 🎯 First-Time Contributors

Looking for something to work on? Check out issues tagged with `good first issue` or `help wanted`. These are specially selected for new contributors!

## 🎉 Fun Facts

- The name "MoraRouter" was inspired by the ancient Roman goddess of speed, Mora (just kidding, we made that up, but it sounds cool!)
- The project was created out of frustration with existing routers that were either too simple or too complex
- The router internals use a clever combination of radix trees and regexp matching for maximum performance
- Some of our contributors have never met in person but collaborate effectively through code!

## ⚖️ License

By contributing, you agree that your contributions will be licensed under the project's MIT License.

---

Thank you for contributing to MoraRouter! Happy coding! 🚀
