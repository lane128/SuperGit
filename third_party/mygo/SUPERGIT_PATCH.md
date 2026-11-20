# SuperGit MyGo compatibility snapshot

Source: github.com/egoist/mygo v0.8.4-0.20261108082913-9233663736ca.
The original license is retained. The dependency version is unchanged.
Website, documentation, examples, benchmarks, frontend workspace and CI files
are omitted; Go library, embedded resources, CLI and automation plugin remain.

Only ui/input.go and ui/runtime.go change runtime behavior. ui/input_barrier_test.go
adds the regression. The reviewable diff is docs/diagnostics/mygo-input-order.patch
in SuperGit. A pending focus transition is committed before the next input arrives;
ordinary typing remains batched, and drawn menus retain their queued key semantics.

SUPERGIT_UPSTREAM.json records every retained file hash. SuperGit's make check
verifies this snapshot and tests the SDK UI package alongside the application.
When upgrading MyGo, replace this snapshot from the new pinned module, reapply or
remove the compatibility diff, update the manifest, and rerun those checks.
