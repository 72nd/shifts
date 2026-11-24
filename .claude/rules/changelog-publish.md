---
paths:
  - "CHANGELOG.md"
  - "internal/web/version.go"
---

# Changelog — format and publishing a version

Publishing is two separate user requests (finalize, then bump). Do not combine them.

**Only bump the version after the user had time to ship the finalized version.**

`internal/web/version.go` (`Version`) stays on the version being shipped until the user asks for the bump.

## Format of `CHANGELOG.md`

- English. One section per release: `## vX.Y.Z – Short summary` (en dash). Oldest first, newest last. Two blank lines between sections.
- `## Feature Request / Backlog` is always the last section. New release headings go **before** it. Only change the backlog when the user asks.
- Bullet format: `- **Type: Short title** One or two sentences.` Allowed types: `Feature`, `Enhancement`, `Fix`, `Refactor`, `Development`. Always bold the `Type: Title` part, always capitalise the type.
- Write for the person using or operating the app. Say what is new or different and where to find it (UI element, API endpoint, `config.toml` key). No empty adjectives ("much better", "robust") unless the same sentence states the concrete behaviour.
- One topic per bullet. New bullets of an open release go at the end of that release's list.
- No bullets for corrections to older changelog entries or other pure doc typo fixes.

## 1. Finalize (user asks to publish / finalize)

1. The highest heading in `CHANGELOG.md` is the open release. Give it its summary: `## vX.Y.Z – Short summary`. That version is already in `version.go`; do not change it.
2. Compare the bullets with `git log <last-tag>..HEAD`. Report missing or wrong entries; add them only if the commits make the content unambiguous.
3. Check that the feature list in `README.md` and `internal/web/docs/help-{de,en}.md` match the release's `Feature` bullets.
4. Run `go vet ./...` and `make build`. If either fails, report it instead of finalizing.
5. Stop. Do **not** add the next heading, bump, commit, tag or push. Tell the user to: commit the finalize, create the tag `vX.Y.Z`, push it, run `make dist` and attach the Linux binaries (amd64, arm64) to the GitHub release. Wait for the bump request.

## 2. Bump (user asks after the tag is pushed)

Only after the user confirms the finalized version is tagged and pushed.

1. Pick the next version from the closed release's bullets:
   - **Minor** (`v0.5.7` → `v0.6.0`, patch resets to 0) for any `Feature` or any change that alters how a familiar task works.
   - **Patch** (`v0.5.7` → `v0.5.8`) for pure fixes, docs, refactors and development changes.
   - If unsure, or the user names a bump, ask or follow the user.
2. Insert an empty `## vX.Y.Z` heading (no summary) before `## Feature Request / Backlog`. Leave the closed release untouched.
3. Set the same version in `internal/web/version.go` (`const Version = "vX.Y.Z"`, with `v` prefix).
4. Add no bullets and do not commit.
