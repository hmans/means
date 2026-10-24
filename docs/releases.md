# Releases

Means uses [Sweet Release 0.2.0](https://github.com/chattocorp/sweet-release/releases/tag/apps%2Fsweet%2Fv0.2.0)
to manage change notes, versions, changelogs, and local tags. GoReleaser builds
binaries and uploads them to a draft GitHub Release when a `v*` tag is pushed.
Maintainers review and publish the draft on GitHub.

Run `mise install` before using these commands. The `mise sweet` task runs the
pinned Sweet version with Go; it does not require a global `sweet` installation.

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

Start from an up-to-date checkout with a clean working tree and full Git history:

```sh
git fetch origin --tags
mise sweet status
mise sweet bump .
git diff
mise test
mise run release:snapshot
```

Sweet updates `.sweet.yml`, writes `CHANGELOG.md`, and removes consumed notes.
The initial manifest is `0.0.0`; the initial feature note prepares `0.1.0`.
Commit the release changes with a Conventional Commit such as
`chore(release): prepare means 0.1.0`, then merge them into `main` through the
normal review process.

From the updated `main` checkout, create and push the release tag:

```sh
git fetch origin --tags
mise sweet git tag --dry-run
mise sweet git tag
git push origin v0.1.0
```

Replace `v0.1.0` with the version being released. Sweet tags the commit that
introduced the version, even if later commits exist. Push only the intended tag.
Do not tag the initial `0.0.0` manifest.

## Build and publish

The release workflow checks that the tag matches `.sweet.yml` and extracts the
matching changelog entry with `sweet changelog`. It runs Go tests and vet, then
builds Linux, macOS, and Windows binaries for amd64 and arm64. Archives use
tar.gz on Linux/macOS and ZIP on Windows, with `checksums.txt` for SHA-256 hashes.
The linker sets the version reported by `means version`.

The workflow uploads a draft named `means <version>`. Review its notes and assets
on GitHub, then publish it. Versions with prerelease suffixes are marked as
prereleases. The workflow uses the built-in `GITHUB_TOKEN` with `contents: write`;
no additional secret is needed.

To run the workflow for an existing tag after the workflow is on `main`:

```sh
gh workflow run release.yml --ref main -f tag=v0.1.0
```

If a failed upload left a partial draft, delete that draft before retrying. Keep
the Git tag. Do not rerun publishing for an already published release.

For local checks, use `mise run release:check` to validate the GoReleaser config
and `mise run release:snapshot` to run tests, vet, and all six builds. Snapshots
write archives into the ignored `dist/` directory and do not publish or create
tags. Their version is `0.0.0-SNAPSHOT-<commit>`; ordinary source builds retain
the `dev` version.
