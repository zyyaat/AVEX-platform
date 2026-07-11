## Summary

<!-- Brief description of what this PR does and why. 1-3 sentences. -->

## Type of change

- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📝 Documentation update
- [ ] ♻️ Refactor (no functional changes, no API changes)
- [ ] ⚡ Performance improvement
- [ ] 🔒 Security fix
- [ ] 🧪 Test addition / improvement
- [ ] 🔧 CI/CD / build infrastructure
- [ ] 🗑️ Chore (cleanup, dependency bump, etc.)

## Affected components

- [ ] Backend (`backend/`)
- [ ] Driver app (`artifacts/driver/`)
- [ ] Customer app (`artifacts/customer/`)
- [ ] Admin app (`artifacts/admin/`)
- [ ] Merchant app (`artifacts/merchant/`)
- [ ] Support app (`artifacts/support/`)
- [ ] CI/CD (`.gitlab-ci.yml` / `.github/workflows/`)
- [ ] Documentation
- [ ] Infrastructure (`docker-compose.yml`, `Dockerfile`)

## Changes

<!-- Describe the key changes in bullet form. -->

-
-
-

## How to test

<!-- How can a reviewer verify this works? -->

1.
2.
3.

## Checklist

- [ ] Branch name follows `<type>/<scope>-<desc>` convention
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
- [ ] `gofmt -w .` and `go vet ./...` are clean (backend changes)
- [ ] `pnpm typecheck` passes in affected app(s) (frontend changes)
- [ ] New code has tests; existing tests still pass
- [ ] No new `console.log` / `fmt.Println` in production code
- [ ] No secrets, tokens, or credentials in the diff
- [ ] `CHANGELOG.md` updated under `[Unreleased]` (user-facing changes)
- [ ] Documentation updated if API or behavior changed
- [ ] Migration file added if schema changed (never edit existing migrations)

## Screenshots / Recordings

<!-- For UI changes, attach before/after screenshots or a screen recording. -->

## Related issues

Closes #
Refs #
