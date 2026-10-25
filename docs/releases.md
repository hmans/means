# Releases

Means uses [Sweet Release 0.3.0](https://github.com/chattocorp/sweet-release/releases/tag/sweet%400.3.0)
to manage change notes, versions, changelogs, and tags. Sweet's reusable GitHub
workflows create release PRs and push version tags. Means owns the separate
tag-triggered workflow that builds archives and publishes GitHub Releases.

Run `mise install` before using these commands. The `mise sweet` task runs the
pinned Sweet version with Go; it does not require a global `sweet` installation.
The Go commands pin the commit behind `sweet@0.3.0`, because that release tag is
not a Go module version tag. The shared workflows build Sweet from their own
revision by default; the local task and release-note extraction use the CLI pin.

## GitHub setup

Add a repository Actions secret named `RELEASE_PR_TOKEN`. Use a fine-grained
personal token with access to `hmans/means`, Contents read/write, and Pull requests
read/write. This token lets release PRs trigger CI and lets pushed tags trigger
the Release workflow. Both shared calls receive it as `release-token`.
The Release workflow uses `GITHUB_TOKEN` to publish archives and notes.

Update the shared workflow references in `bump.yml` and `ci.yml` together.
Use a merged commit SHA or an immutable `workflows-vX.Y.Z` tag. This draft
temporarily pins the reviewed Sweet PR commit; replace that pin after
chattocorp/sweet-release#47 merges and before merging this integration.

## Record a change

Run `mise sweet change .` and commit the generated `.sweet/*.md` file with the
code. For non-interactive use:

```sh
printf 'Describe the user-visible change.\n' | mise sweet change . --type fix --stdin
mise sweet status
```

Features bump the minor version. Fixes and performance changes bump the patch
version. Chores and refactors do not cause a release. Set `breaking: true` in a
note's frontmatter for breaking changes; these require a minor bump before 1.0
and a major bump after that. Policies live in `.sweet.config.yml`.

## Prepare a release

Every push to `main` starts the shared `bump.yml` workflow. It updates `.sweet.yml`,
writes `CHANGELOG.md`, consumes change notes, and opens or updates a release PR.
Review and merge that PR through the normal review process. With no releasable
notes, no release PR is needed.

CI runs Go race tests and vet for PRs and `main` pushes. After checks pass on
`main`, the shared `tag.yml` workflow examines that push's commit range and
creates tags for committed version changes. Main runs are not canceled by newer
pushes. Root-package tags retain the `v<version>` format.
PRs run checks only; tagging runs only on pushes to `main`.

For a local preview, use a clean checkout with full Git history:

```sh
git fetch origin --tags
mise sweet status
mise sweet bump .
git diff
mise test
mise run release:snapshot
```

The bump commands modify the checkout. Use a separate worktree for previews;
let the automated PR own release preparation on `main`. Tags point to the commit
that introduced the version, even if later commits exist. Existing versions are
not released again just because the automation is installed.

## Build and publish

The repository's `.github/workflows/release.yml` runs on `v*` tag pushes and extracts
the matching changelog entry with `sweet changelog`. GoReleaser runs tests and vet, then
builds Linux, macOS, and Windows binaries for amd64 and arm64. Archives use
tar.gz on Linux/macOS and ZIP on Windows, with `checksums.txt` for SHA-256 hashes.
The linker sets the version reported by `means version`.

The workflow verifies the Linux binary version and all archive checksums, then
uses `gh release create --verify-tag` to create a draft named `means <version>`,
upload archives and checksums, and publish. Build or upload failures leave the release
unpublished. Versions with prerelease suffixes are marked as
prereleases. The workflow uses the built-in `GITHUB_TOKEN` with `contents: write`;
release PR creation and tag pushes use `RELEASE_PR_TOKEN` as described above.

If release PR preparation fails, rerun all jobs in the Bump run. For a failed
tagging job, rerun the original CI run to preserve its commit range. Matching
tags are kept and missing tags are pushed. For failed publication, rerun the
tag's Release run. It rebuilds the archives, recreates any incomplete draft,
and skips publication if the release is already published. Other tags' releases
are untouched.

For local checks, use `mise run release:check` to validate the GoReleaser config
and `mise run release:snapshot` to run tests, vet, and all six builds. Snapshots
write archives into the ignored `dist/` directory and do not publish or create
tags. Their version is `0.0.0-SNAPSHOT-<commit>`; ordinary source builds retain
the `dev` version.
