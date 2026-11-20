# Complete Summary Reading

**Status:** Implemented; targeted race, independent review, whole ordinary suite and vet passed. No commit, installation or publication.

**Goal:** Make the entire historical commit subject readable and selectable in the existing summary, including narrow panes and maximum font scale.

## Finding and decision

The independent design role, using Emil Design Engineering and Impeccable, found that `commitInfoContent` limited subjects to three rows while `commitBody` removes the subject from the body. Thus the omitted subject tail could not be reached by scrolling. This is a reading defect in the current capability, not an additional visual setting or control. Remove only the subject's `Rows: 3`; keep the existing selectable text and containing scroll area.

Product resource research also found `open_context_url` in the installed Sublime Merge Commit Message menu. It has no explicit arguments/caption, and context positioning/protocol rules remain unverified. Existing full-message and subject copying already exist; do not count them as missing. URL opening remains a candidate, not an implemented feature in this change.

## Evidence and implementation

`commit_subject_reading_test.go` renders a long subject at 292×260 with actual maximum scale 1.15, under a clickable ancestor and a bounded real scroll area. The original three-row cap failed with a 60.375px title block. Removing the cap permits full height; the test scrolls to MaxY, checks the final line lies inside the viewport, drags that line and copies it through Cmd/Ctrl+C, asserting the unique `SUBJECT_END` suffix. This proves selectable tail text, not merely an accessible full string.

The initial fixture omitted sizing the ScrollArea return element and therefore had no scroll range; corrected the fixture with `.Fill()` per SDK sizing contract. No additional production scroll change was needed. An overlay restoring the three-row cap against the final retained test failed as expected. Combined long-subject, body selection, date-menu and author-menu race passed 2.830 s. The independent design-role second review read the final implementation and regression test and found no concrete regression or false-positive assertion; did not rerun tests.

The date lifecycle follow-up also uses one tester to verify initial clipboard failure, refreshed author copy, then reopened date retry with the refreshed value. Exact target sequence and error/success feedback are asserted. Previous date ordinary/vet acceptance at 511 inputs remains separate.

## Full check and limits

Whole ordinary suite and vet passed as `bvtuger3j`, prefix `build/alignment-summary-reading`. Main read complete log, source verification and task output: 512 source inputs unchanged, eight packages matched, check/diff exits zero; lobedesk ordinary 147.154 s, other packages cached. Full UI race from the earlier 508-input author identity snapshot is not reused as full-race evidence for this later fix; focused race covers the changed summary paths.

A subsequent scratch overlay reran the retained end-of-subject selection scenario and captured `build/long-subject-tail.png` (0.505 s). Main read the image: the final `SUBJECT_END` line is visible and selected inside the narrow viewport, with the scroll position at the bottom. No production source changed for this capture.

Headless interaction and local rendering do not establish native OS clipboard, reference behavior parity, or Windows/Linux runtime. No native permissions were bypassed. Date-menu English/light and Chinese/dark captures at 480×360, maximum font and long author were generated and read in `build/date-menu-captures/`; that evidence is recorded in copy-date.
