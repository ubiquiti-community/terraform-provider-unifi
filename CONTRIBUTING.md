# Contributing

Thanks for contributing to the UniFi Terraform provider. Build and test with
`go build ./...` and `go test ./...` (acceptance tests: `make testacc`); see
`.github/copilot-instructions.md` for the provider's development conventions.

## Changelog

Do not edit `CHANGELOG.md`. Each PR adds its own changelog fragment, a small
YAML file under `.changes/unreleased/`, and the fragments are assembled into
`CHANGELOG.md` when a release is cut. PRs therefore never conflict on the
changelog.

Fragments are managed with [changie](https://changie.dev), pinned as a Go tool
in `go.mod` (no install needed):

```sh
go tool changie new
```

It asks for the kind and a one-line body, and writes
`.changes/unreleased/<kind>-<timestamp>.yaml`. Commit that file with your
change. You can also pass both on the command line
(`go tool changie new --kind fix --body '...'`) or write the file by hand:

```yaml
# .changes/unreleased/fix-20261031-120000.yaml
kind: fix
body: '**`unifi_wan`: `egress_qos.priority` is no longer sent as `0`.** The controller rejected the whole WAN update; an unset priority is now left unset (#678, from go-unifi#105 by @florinsule)'
time: 2026-10-31T12:00:00Z
```

| `kind`        | Section in CHANGELOG.md   |
| ------------- | ------------------------- |
| `feature`     | `### ✨ Features`          |
| `breaking`    | `### ⚠️ Breaking Changes`  |
| `deprecation` | `### ⚠️ Deprecations`      |
| `fix`         | `### 🐛 Bug Fixes`         |
| `docs`        | `### 📚 Documentation`     |
| `maintenance` | `### 🔧 Maintenance`       |
| `dependency`  | `### 🔧 Dependency Updates` |

Write the body like the existing entries: a bold one-line summary naming the
resource, then what changed and why, then the references in parentheses:
`(#PR, from #issue by @user)`. If you don't know the PR number yet, add it
after opening the PR or leave it to the maintainer. One fragment per
user-visible change; a PR with a feature and a fix adds two. A body that needs
nested bullets can use a YAML block, with the nested lines indented two spaces
more than the first line:

```yaml
kind: breaking
body: |-
  **Prefixed attributes are now nested objects.** The mapping:
    - `unifi_device`: `led_override` → `led.override`
    - `unifi_wan`: `wan_dslite_remote_host` → `dslite.remote_host`
```

The **Changelog** check fails when a PR changes provider code (non-test Go
files under `unifi/`, or `main.go`) without adding a fragment. It is not a
required check. A maintainer can add the `skip-changelog` label for changes
users won't notice (refactors, tests, CI). Dependabot PRs are skipped: the
maintainer summarises dependency bumps at release time.

## Releasing (maintainers)

`.changes/vX.Y.Z.md` holds the released notes of each version, and
`CHANGELOG.md` is generated from them by `go tool changie merge` (the
**CHANGELOG.md in sync** check verifies it). To fix the notes of an old
release, edit its `.changes/vX.Y.Z.md` and run `go tool changie merge`.

1. From an up-to-date `main`, create a release branch.
2. Add fragments for anything that has none yet, typically the dependency
   bumps merged since the last release:
   `go tool changie new --kind dependency --body 'go-unifi v1.42.0 → v1.43.0 (#700)'`.
3. Optional: write the release intro and upgrade notes in
   `.changes/unreleased/header.md`. It is rendered verbatim under the version
   heading, before the sections, and removed by the batch. The file is a Go
   template, so it must not contain `{{`.

   ```markdown
   One paragraph summarising the release. go-unifi is now v1.43.0.

   **Upgrade notes:**

   - `unifi_wan`: what users must know or change before upgrading.
   ```

4. Check the next version (`breaking` and `feature` bump the minor version,
   the other kinds the patch version), then batch and merge:

   ```sh
   go tool changie next auto            # e.g. v0.64.0
   go tool changie batch auto           # or: go tool changie batch v0.64.0
   go tool changie merge
   ```

   `batch` writes `.changes/v0.64.0.md` (heading `## [v0.64.0] - YYYY-MM-DD`
   with today's date) and deletes the fragments and `header.md`; `merge`
   rebuilds `CHANGELOG.md`. Review the new section, edit
   `.changes/v0.64.0.md` if needed and run `go tool changie merge` again.
   `git diff CHANGELOG.md` must only add the new section at the top.
5. Commit as `chore(release): v0.64.0`, open the PR and merge it.
6. Tag the merge commit (`v0.64.0`) and push the tag: the **Release Terraform
   Provider** workflow publishes the release with GoReleaser.
7. Once the release exists, run the **Update Release Notes** workflow
   (`gh workflow run update-release-notes.yaml`) to copy the CHANGELOG.md
   section into the GitHub release.
