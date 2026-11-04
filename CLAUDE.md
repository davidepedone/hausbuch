# hausbuch

A private household wiki: the place where the people living in a home collect everything they may need to know —
recipes, how-to procedures, seasonal chores, where things are kept.

## Workflow

Spec-driven, with OpenSpec (`openspec/`):

- `openspec/specs/` is the source of truth for what the system does today.
- Every change starts as a proposal in `openspec/changes/<name>/` (`/opsx:propose`), is implemented only after the
  plan is agreed (`/opsx:apply`), and is archived when merged (`/opsx:archive`), which folds its spec deltas into
  `openspec/specs/`.
- If implementation shows the spec is wrong, fix the change's artifacts first, then the code.
- Tests are derived from the spec scenarios: every scenario of an implemented requirement has a test.

## Constraints

- The product is a generic Docker image. Nothing in this repo refers to a specific installation: configuration
  comes from environment variables, documented in the README.
- Every page and endpoint requires login, except the login page itself. Being on the home network grants nothing.
- Accounts are managed with the binary's `user` commands, run inside the container; there is no self-registration
  and no account management in the web interface.
- UI, code, comments, commit messages and docs are in English.
- The UI is responsive: it must work well on desktop, tablet and phone.
