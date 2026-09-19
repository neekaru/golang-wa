# Repository Instructions

## Validation

- Do not run builds, test suites, smoke tests, containers, or services unless the user explicitly requests them in the current conversation.
- This includes `go test`, `go build`, Docker builds, and runtime launch checks. They are slow and costly in this repository.
- Prefer source inspection, upstream patch analysis, LSP diagnostics, and other non-executing static checks.
- Do not use a successful build as proof that WhatsApp or Whatsmeow runtime behavior is correct.
- When runtime verification would normally be useful but was not explicitly requested, state that it was not run instead of starting it.
