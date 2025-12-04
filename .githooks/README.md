# Git Hooks

This directory contains Git hooks for the project to ensure code quality.

## Installation

To install the hooks, run from the project root:

```bash
git config core.hooksPath .githooks
```

Or use the make target (if available):

```bash
make install-hooks
```

## Available Hooks

### pre-commit

Runs before every commit and checks:
- ✅ **go fmt** - ensures code is properly formatted
- ✅ **go vet** - runs static analysis to find potential issues
- ✅ **go test** - runs all tests to ensure nothing is broken
- ✅ **go mod tidy** - ensures go.mod and go.sum are clean

If any check fails, the commit will be aborted.

## Bypassing Hooks (Use with Caution!)

If you need to bypass the hooks (NOT recommended), use:

```bash
git commit --no-verify
```

⚠️ **Warning**: Only bypass hooks if you know what you're doing!

## Disabling Hooks

To disable hooks temporarily:

```bash
git config core.hooksPath .git/hooks
```

To re-enable:

```bash
git config core.hooksPath .githooks
```
