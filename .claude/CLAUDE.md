# CLAUDE.md

## Code style

- Go 1.27.+, standard conventions

## Behavior Rules

- Do not make Git commits unless explicitly requested.
- When reporting information to me, be extremely concise and sacrifice grammar for the sake of concision.
- In code: default to writing no comments. Never write multi-paragraph docstrings or multi-line comment blocks — one short line max.

## Verification

- Run `make verify` after every code change. Do not report a task as done until it passes.
- On failure, fix the implementation. Never modify, delete, or skip existing tests to make them pass. If a test itself seems wrong, stop and ask me.
- Use `make fmt` to fix formatting failures.

## API Contract

- `openapi.yaml` is the API contract and source of truth.
- Never modify it to make tests pass. If a task requires a contract change, stop and ask me first.

## Language Rules

- Use English for all code, inline comments, and commit messages.
- Use Traditional Chinese (zh-TW) for all conversational responses, explanations, and summaries.
- Keep technical terms and proper nouns in their original language (usually English).