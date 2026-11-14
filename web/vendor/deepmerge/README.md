# Domus-patched deepmerge

This is a local, dependency-free copy of the MIT-licensed `deepmerge` 4.3.1
implementation. Domus applies the upstream remediation proposed in
[TehShrike/deepmerge#274](https://github.com/TehShrike/deepmerge/pull/274):
`__proto__` keys are rejected in both the target-copy and source-copy loops.

The local package version is `4.3.2+domus.1` so it satisfies consumers'
`^4.2.2` range and is distinguishable from the unpatched registry release.
The upstream package has no published patched release. Keep this copy until
upstream publishes and verifies a release containing the fix.

Regression coverage is in `web/src/security/deepmerge-security.test.ts`.

Upstream project: https://github.com/TehShrike/deepmerge
License: MIT (see LICENSE).
