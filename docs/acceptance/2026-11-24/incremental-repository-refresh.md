# Incremental repository refresh — native acceptance

Environment: macOS arm64, Go 1.27.4, Apple Git 2.54.0. Development bundle 0.1.0 (51), built with `make app`, signature verified. Launched the bundle against an isolated checkout; it remained running and read refs. Interaction timings below use the native `make lobe-e2e-build` app and mygo automation, not an internal benchmark.

## Automated checks

The performance work's `go test -cover ./... && make check` run exited 0, including the ordinary suite, project race suite, MyGo UI race checks and `go vet`. Later sidebar-default and Fetch-tab interaction changes were checked separately with targeted tests and vet, not a second full coverage run.

## Follow-up tab interaction checks

Branches start collapsed and remain manually expandable. Activation checks for an already loaded checkout keep the existing view and suppress local-validation progress. Manual Fetch and configured automatic Fetch remain available.

A native reproduction using three existing checkouts confirmed that activating a still-loading destination disabled subsequent tab switches. The fix permits navigation while per-checkout reads finish, settles inactive validation callbacks, and retains write and startup-restore guards. Closing an idle inactive checkout now acts directly without activation, including while the active checkout fetches; unsent draft confirmation remains required. Target-local Fetch/write operations still block closing that target.

Targeted `Tab|Fetch|Confirm|RefreshProgress` tests, `go vet ./...`, `gofmt` and `git diff --check` passed. Native mouse clicks in light 1320×700 and dark 900×640 windows verified switching without initiating Fetch and directly closing an inactive checkout during a deliberately held manual Fetch. The network hold was a test wrapper; this check exercises UI behavior, not remote connectivity. The latest bundle was installed and verified locally at the user's request.

## Window and cancellation checks

Final two-remote fixture passed in dark 900×640 and light 1320×700 windows. Visually checked both native workspace screenshots: the 75-module count and modified/initialized labels were legible. Each run retained the first remote's update after cancellation on the second remote. Screenshots and command logs are local under `build/incremental-refresh-247p8l3o/` (dark) and `build/incremental-refresh-sn36g9oo/` (light).

## SoulApp: five samples (seconds)

Command: `python3 -I scripts/native-lobe-incremental-refresh-performance.py --repo /Users/adam/SoulApp`. Five successful Fetch operations, 75 configured child modules. The script verifies unchanged parent HEAD/index entries per run. Independent review resolved both correctness findings (child error recovery and history selection) before final acceptance. Raw local evidence: `build/incremental-refresh-30krvgzk/samples.json`.

| Sample | First usable parent | All child rows settled | After Fetch process exit to tracking/Sync UI settled |
|---|---:|---:|---:|
| 1 | 3.572 | 5.451 | 0.377 |
| 2 | 2.684 | 4.520 | 0.479 |
| 3 | 2.678 | 4.676 | 0.454 |
| 4 | 2.761 | 4.725 | 0.494 |
| 5 | 2.498 | 4.348 | 0.477 |
| Median | 2.684 | 4.676 | 0.477 |
| Maximum | 3.572 | 5.451 | 0.494 |

Both machine-specific targets met: first usable parent <=4 s median, post-Fetch <=1 s median. All five post-Fetch watcher windows had zero `ls-files`, `ls-tree` or `git submodule` commands. Normal status polling still runs. These are warm local measurements, not cold-cache or network throughput guarantees. Baseline: one prior native trace reached automatic Fetch around 14 s, then spent roughly 13 s refreshing after the remote command. Its start-of-Fetch marker is not identical to the new interactive UI assertion.

Isolated 75-child fixture prior to final review fixes: five-run medians 1.248 s first usable, 2.015 s all children, 0.246 s post-Fetch. Final real-repo numbers above are from the corrected build.
