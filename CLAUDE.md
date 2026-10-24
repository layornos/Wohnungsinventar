## Code language

Everything in code is English: identifiers, functions, types, comments, API paths, JSON fields, database tables and columns, CSS classes, CLI commands, test names and commit messages. Domain terms use the fixed English names from the "Code names" table in `CONTEXT.md`. Only user-facing text in the UI (and specs/issues under `.scratch/`) stays German.

## Agent skills

### Issue tracker

Issues live as local markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
