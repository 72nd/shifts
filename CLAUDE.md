# Shifts (schipla)

Collaborative shift planning web app (Go module `github.com/72nd/shifts`). Participants sign up with just a link and a nickname (no login, no email); organisers manage a plan via an admin password. One instance hosts any number of independent shift plans, each reachable via a non-guessable token URL. Branch `v2` is the development branch; releases are git tags.

The original design brief is `opt/v2-grundprompt.txt` (German). Read it before making architectural decisions; the points below are its essence.

## Design principles (keep these)

- **KISS.** The feature set is essentially final. No speculative abstractions; code must stay quickly understandable for humans.
- **Fixed stack:** Go, `gorilla/mux`, `go-sqlite3` (needs CGO), `bluemonday` for sanitising, `slog` for logging, config in `config.toml` (`config.template.toml` is the reference; `config.toml` is gitignored).
- **Frontend:** Alpine.js stays in charge of rendering via HTML templating; JS models hold state and client/server communication. Tailwind + daisyUI. **No npm**, neither for development nor operation; third-party assets are vendored in `internal/web/static/vendor/`. Reusable UI pieces are Go `html/template` partials. Use semantic HTML elements or ARIA for accessibility.
- **i18n:** Alpine i18n plugin with flat JSON files `internal/web/static/i18n/{de,en}.json` (keep them flat). User-entered fields that need both languages (e.g. titles) must be filled in both.
- **Language:** code, comments, log output, commit messages and the changelog are English. German UI text uses gender-neutral wording, otherwise a colon (`Teilnehmer:innen`), and addresses the user informally with "Du".

## Layout and conventions

- `cmd/shifts/main.go`: entry point. `internal/`: `config`, `db`, `export` (CSV/JSON/PDF via Typst), `model`, `util`, `validation`, `web`.
- All data structures live in `internal/model`; DB types are never exposed through the API.
- DB errors are typed sentinel errors in `internal/db/errors.go`, not string matching. All DB operations run in a transaction.
- Validation is centralised in `internal/validation`. Shared HTTP handler logic belongs in `internal/web/helpers.go`.
- API is documented with swag annotations on handlers.
- In-app help is Markdown in `internal/web/docs/help-{de,en}.md` (change both languages together); see `DOCUMENTATION_SYSTEM.md`. When adding user-visible features, update the feature list in `README.md` too.

## Commands

- `make help` lists targets. `make install` generates Swagger docs and builds; `make build` only builds (`CGO_ENABLED=1`); `make dist` cross-compiles Linux binaries (needs cross compilers).
- `go vet ./...` for static checks.
- There are **no Go tests**. `load-test/` holds a Locust script. Verify changes with `go vet ./...`, `make build` and by running `./shifts` (port from `config.toml`) and exercising the feature.

## Do not touch / do not read

- `docs/` (Swagger output, gitignored and generated): never edit by hand; change handler annotations and run `make docs`.
- `v1/`: legacy implementation, reference only.
- `data/`, `local/`, `config.toml`, `SECRET.txt`, `12w-deploy.md`: private runtime data, secrets and deployment notes. Do not read or print them.
- `opt/`: design assets and notes. `opt/specStory.md` is a ~400 KB chat log; do not read it.

## Releases

Changelog format and the finalize/bump workflow are in `.claude/rules/changelog-publish.md`. Never bump the version, tag or push unless asked.
