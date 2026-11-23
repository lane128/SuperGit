package lobedesk

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"supergit/internal/updater"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const updateStartupDelay = 10 * time.Second

type updateController struct {
	mgr                      *updater.Manager
	dialogOpen               bool
	dialogOwner              toastOwner
	manualPending            bool
	manualOwner              toastOwner
	promptedVersion          string
	pendingAutoPromptVersion string
	notesScroll              ui.ScrollState
	quitFn                   func()
}

func (a *app) updateCtrl() *updateController {
	if a.repositoryTabs != nil {
		return a.repositoryTabs.updater
	}
	return a.updateState
}

func (a *app) ensureUpdater() *updateController {
	if ctrl := a.updateCtrl(); ctrl != nil {
		return ctrl
	}
	short, build := currentAppVersionParts()
	ctrl := &updateController{}
	cfg := updater.Config{
		CurrentVersion: short,
		CurrentBuild:   build,
		AutoCheck:      a.autoCheckUpdates,
		AutoDownload:   a.autoDownloadUpdates,
		SkippedVersion: a.skippedUpdateVersion,
		LastCheckedAt:  a.lastUpdateCheckAt,
	}
	ctrl.attachManager(a, cfg)
	if a.repositoryTabs != nil {
		a.repositoryTabs.updater = ctrl
	} else {
		a.updateState = ctrl
	}
	return ctrl
}

func (ctrl *updateController) attachManager(ownerApp *app, cfg updater.Config) {
	cfg.OnChange = func(snap updater.Snapshot) {
		target := ownerApp.activeRepositorySession()
		target.hopToMain(func() {
			ctrl.handleSnapshot(target.activeRepositorySession(), snap)
		})
	}
	ctrl.mgr = updater.NewManager(cfg)
}

func (a *app) initUpdater() *updateController {
	return a.ensureUpdater()
}

func (a *app) watchUpdates() {
	if strings.TrimSpace(os.Getenv(updater.EnvDisableAutoUpdate)) != "" {
		return
	}
	ctrl := a.ensureUpdater()
	ctx := a.ctx
	if a.repositoryTabs != nil && a.repositoryTabs.ctx != nil {
		ctx = a.repositoryTabs.ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(updateStartupDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
	ctrl.mgr.CheckIfDue(ctx)

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctrl.mgr.CheckIfDue(ctx)
		}
	}
}

func (a *app) updateSnapshot() updater.Snapshot {
	if ctrl := a.updateCtrl(); ctrl != nil && ctrl.mgr != nil {
		return ctrl.mgr.Snapshot()
	}
	short, build := currentAppVersionParts()
	return updater.Snapshot{
		State:          updater.StateIdle,
		CurrentVersion: short,
		CurrentBuild:   build,
		AutoCheck:      a.autoCheckUpdates,
		AutoDownload:   a.autoDownloadUpdates,
		SkippedVersion: a.skippedUpdateVersion,
		LastCheckedAt:  a.lastUpdateCheckAt,
	}
}

func (a *app) syncUpdaterPreferences() {
	if a.repositoryTabs != nil {
		for _, session := range a.repositoryTabs.sessions {
			if session == a {
				continue
			}
			session.autoCheckUpdates = a.autoCheckUpdates
			session.autoDownloadUpdates = a.autoDownloadUpdates
			session.skippedUpdateVersion = a.skippedUpdateVersion
			session.lastUpdateCheckAt = a.lastUpdateCheckAt
		}
	}
	if ctrl := a.updateCtrl(); ctrl != nil && ctrl.mgr != nil {
		ctrl.mgr.SetPreferences(a.autoCheckUpdates, a.autoDownloadUpdates, a.skippedUpdateVersion)
	}
}

func (a *app) setAutoDownloadPreference(enabled bool, owner toastOwner) {
	if a.autoDownloadUpdates == enabled && (!enabled || a.autoCheckUpdates) {
		return
	}
	a.autoDownloadUpdates = enabled
	if enabled {
		a.autoCheckUpdates = true
	}
	a.syncUpdaterPreferences()
	a.saveSettingsFor(owner)
	a.invalidate()
}

func (a *app) setAutoCheckPreference(enabled bool, owner toastOwner) {
	if a.autoCheckUpdates == enabled {
		return
	}
	a.autoCheckUpdates = enabled
	a.syncUpdaterPreferences()
	a.saveSettingsFor(owner)
	a.invalidate()
}

func (ctrl *updateController) handleSnapshot(a *app, snap updater.Snapshot) {
	if !snap.LastCheckedAt.IsZero() && !snap.LastCheckedAt.Equal(a.lastUpdateCheckAt) {
		a.lastUpdateCheckAt = snap.LastCheckedAt
		a.syncUpdaterPreferences()
		a.saveSettingsFor(toastMain)
	}

	if ctrl.manualPending && snap.State != updater.StateChecking {
		owner := ctrl.manualOwner
		ctrl.manualPending = false
		switch snap.State {
		case updater.StateUpToDate:
			a.queueFor(owner, "info", a.tf("updateUpToDateNotice", formatVersionLabel(snap.CurrentVersion, snap.CurrentBuild)), "")
		case updater.StateAvailable, updater.StateDownloading, updater.StateVerifying, updater.StateReadyToRestart:
			a.openUpdateDialog(owner)
		case updater.StateError:
			a.queueFor(owner, "error", a.t("updateCheckFailedTitle"), snap.Error)
		}
	}

	if snap.State == updater.StateReadyToRestart && snap.AutoDownloadOrigin && snap.Release.Version != "" && ctrl.promptedVersion != snap.Release.Version {
		if !ctrl.dialogOpen && !a.hasPendingDialog(toastMain) {
			ctrl.promptedVersion = snap.Release.Version
			ctrl.pendingAutoPromptVersion = ""
			ctrl.dialogOpen = true
			ctrl.dialogOwner = toastMain
		} else if !ctrl.dialogOpen {
			ctrl.pendingAutoPromptVersion = snap.Release.Version
		}
	}
	a.invalidate()
}

func (a *app) maybeOpenDeferredUpdatePrompt() {
	ctrl := a.updateCtrl()
	if ctrl == nil || ctrl.dialogOpen || ctrl.pendingAutoPromptVersion == "" {
		return
	}
	snap := ctrl.mgr.Snapshot()
	if snap.State != updater.StateReadyToRestart || snap.Release.Version != ctrl.pendingAutoPromptVersion {
		ctrl.pendingAutoPromptVersion = ""
		return
	}
	if a.hasPendingDialog(toastMain) {
		return
	}
	ctrl.promptedVersion = snap.Release.Version
	ctrl.pendingAutoPromptVersion = ""
	ctrl.dialogOpen = true
	ctrl.dialogOwner = toastMain
	a.invalidate()
}

func (a *app) updateDialogOpenFor(owner toastOwner) bool {
	ctrl := a.updateCtrl()
	return ctrl != nil && ctrl.dialogOpen && ctrl.dialogOwner == owner
}

func (a *app) hasVisibleUpdate() bool {
	snap := a.updateSnapshot()
	switch snap.State {
	case updater.StateAvailable, updater.StateDownloading, updater.StateVerifying, updater.StateReadyToRestart:
		return snap.Release.Version != ""
	default:
		return false
	}
}

func (a *app) checkForUpdates(owner toastOwner) {
	ctrl := a.ensureUpdater()
	snap := ctrl.mgr.Snapshot()
	if snap.State == updater.StateDownloading || snap.State == updater.StateVerifying || snap.State == updater.StateReadyToRestart {
		a.openUpdateDialog(owner)
		return
	}
	if snap.State == updater.StateChecking {
		ctrl.manualPending = true
		ctrl.manualOwner = owner
		a.invalidate()
		return
	}
	ctrl.manualPending = true
	ctrl.manualOwner = owner
	ctx := a.ctx
	if a.repositoryTabs != nil && a.repositoryTabs.ctx != nil {
		ctx = a.repositoryTabs.ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctrl.mgr.CheckNow(ctx, true)
	a.invalidate()
}

func (a *app) openUpdateDialog(owner toastOwner) {
	ctrl := a.ensureUpdater()
	if ctrl.dialogOpen && ctrl.dialogOwner == owner {
		a.invalidate()
		return
	}
	if a.rejectPendingDialog(owner) {
		return
	}
	ctrl.dialogOpen = true
	ctrl.dialogOwner = owner
	if snap := ctrl.mgr.Snapshot(); snap.Release.Version != "" && snap.State == updater.StateReadyToRestart {
		ctrl.promptedVersion = snap.Release.Version
		ctrl.pendingAutoPromptVersion = ""
	}
	a.invalidate()
}

func (a *app) closeUpdateDialog() {
	if ctrl := a.updateCtrl(); ctrl != nil && ctrl.dialogOpen {
		ctrl.dialogOpen = false
		a.invalidate()
	}
}

func (a *app) startUpdateDownload() {
	ctrl := a.ensureUpdater()
	ctx := a.ctx
	if a.repositoryTabs != nil && a.repositoryTabs.ctx != nil {
		ctx = a.repositoryTabs.ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctrl.mgr.StartDownload(ctx)
	a.invalidate()
}

func (a *app) skipAvailableUpdate(owner toastOwner) {
	ctrl := a.updateCtrl()
	if ctrl == nil || ctrl.mgr == nil {
		return
	}
	skipped := ctrl.mgr.SkipAvailable()
	ctrl.dialogOpen = false
	ctrl.pendingAutoPromptVersion = ""
	if skipped != "" {
		a.skippedUpdateVersion = skipped
		a.syncUpdaterPreferences()
		a.saveSettingsFor(owner)
		a.queueFor(owner, "info", a.tf("updateSkippedNotice", "v"+skipped), "")
	}
	a.invalidate()
}

func (a *app) sessionsForRestartCheck() []*app {
	if a.repositoryTabs != nil && len(a.repositoryTabs.sessions) > 0 {
		return a.repositoryTabs.sessions
	}
	return []*app{a}
}

func (a *app) restartBlockerKey() string {
	for _, session := range a.sessionsForRestartCheck() {
		if session.busy || session.loading || session.fetching {
			return "updateBusyBeforeRestart"
		}
		for _, ed := range session.conflictEditors {
			if ed != nil && ed.Loaded && ed.Text != ed.Original.Working {
				return "updateConflictBeforeRestart"
			}
		}
	}
	return ""
}

func (a *app) applyPreparedUpdate(owner toastOwner) {
	ctrl := a.updateCtrl()
	if ctrl == nil || ctrl.mgr == nil {
		return
	}
	if blocker := a.restartBlockerKey(); blocker != "" {
		a.queueFor(owner, "warning", a.t(blocker), "")
		a.invalidate()
		return
	}
	if a.repositoryTabs != nil {
		a.repositoryTabs.saveSession()
	}
	if err := ctrl.mgr.PrepareApply(os.Getpid()); err != nil {
		a.queueFor(owner, "error", a.t("updateApplyFailed"), err.Error())
		a.invalidate()
		return
	}
	ctrl.dialogOpen = false
	quit := ctrl.quitFn
	if quit == nil {
		quit = mygo.App.Quit
	}
	quit()
}

func (a *app) updateBadgeText(snap updater.Snapshot) (string, *ui.SVG, bool, bool) {
	ver := "v" + snap.Release.Version
	switch snap.State {
	case updater.StateAvailable:
		return a.tf("updateBadgeAvailable", ver), lucide.ArrowUpCircle(), false, false
	case updater.StateDownloading:
		if snap.TotalBytes > 0 {
			pct := int(min(100, (snap.DownloadedBytes*100)/snap.TotalBytes))
			return a.tf("updateBadgeDownloading", pct), lucide.LoaderCircle(), true, false
		}
		return a.t("updateBadgeDownloadingBare"), lucide.LoaderCircle(), true, false
	case updater.StateVerifying:
		return a.t("updateBadgeVerifying"), lucide.LoaderCircle(), true, false
	case updater.StateReadyToRestart:
		return a.tf("updateBadgeReady", ver), lucide.RotateCw(), false, true
	default:
		return "", nil, false, false
	}
}

// updateTitleBarBadge renders an unobtrusive pill in the window chrome only
// while an update is available, downloading, verifying, or ready to restart.
func (a *app) updateTitleBarBadge(c *ui.Context) {
	a.maybeOpenDeferredUpdatePrompt()
	snap := a.updateSnapshot()
	text, iconSVG, spinning, ready := a.updateBadgeText(snap)
	if text == "" {
		return
	}
	tk := theme.TokensOf(c)
	bg, fg, border := updateBadgeColors(tk, ready, false)
	btn := ui.ButtonBase(c.Key("update-titlebar-badge")).
		Height(scaledFont(26)).PaddingX(10).Gap(5).Radius(13).Shrink(0).
		AlignItems(ui.Center).Justify(ui.Center).
		Label(text).Tooltip(a.t("updateBadgeTooltip"))
	if btn.Hovered() {
		bg, fg, border = updateBadgeColors(tk, ready, true)
	}
	btn.Background(bg).TextColor(fg).Border(1, border)
	btn.Children(func() {
		ic := ui.Icon(c, iconSVG).FontSize(scaledFont(13)).TextColor(fg)
		if spinning {
			ic.Rotate(ic.Loop("spin", 900*time.Millisecond, ui.Linear) * 360)
		}
		ui.Text(c, text).FontSize(scaledFont(11.5)).FontWeight(600).TextColor(fg).SingleLine()
	})
	if btn.Clicked() {
		a.openUpdateDialog(toastMain)
	}
}

func updateBadgeColors(tk *theme.Tokens, ready, hovered bool) (bg, fg, border ui.Color) {
	if ready {
		if tk.Dark {
			if hovered {
				return ui.Hex("#1E442C"), ui.Hex("#56D364"), ui.Hex("#3FB950").Alpha(0.45)
			}
			return ui.Hex("#163522"), ui.Hex("#3FB950"), ui.Hex("#3FB950").Alpha(0.32)
		}
		if hovered {
			return ui.Hex("#DAFBE1"), ui.Hex("#116329"), ui.Hex("#1A7F37").Alpha(0.35)
		}
		return ui.Hex("#EAF7EE"), ui.Hex("#1A7F37"), ui.Hex("#1A7F37").Alpha(0.24)
	}
	if tk.Dark {
		if hovered {
			return ui.Hex("#1F3654"), ui.Hex("#A5D6FF"), ui.Hex("#58A6FF").Alpha(0.45)
		}
		return ui.Hex("#1B2D45"), ui.Hex("#79C0FF"), ui.Hex("#58A6FF").Alpha(0.32)
	}
	if hovered {
		return ui.Hex("#DDF4FF"), ui.Hex("#0550AE"), ui.Hex("#0969DA").Alpha(0.35)
	}
	return ui.Hex("#EDF4FF"), ui.Hex("#0969DA"), ui.Hex("#0969DA").Alpha(0.24)
}

func (a *app) updateIneligibleMessage(snap updater.Snapshot) string {
	if !snap.Eligibility.Eligible {
		switch snap.Eligibility.Reason {
		case updater.ReasonTranslocated:
			return a.t("updateIneligibleTranslocated")
		case updater.ReasonNotWritable:
			return a.t("updateIneligibleNotWritable")
		case updater.ReasonUnsupportedOS:
			return a.t("updateIneligibleUnsupported")
		default:
			return a.t("updateIneligibleNotApp")
		}
	}
	if !snap.HasPublicKey {
		return a.t("updateIneligibleNoKey")
	}
	if !snap.Release.HasAsset {
		return a.t("updateIneligibleNoAsset")
	}
	return ""
}

func (a *app) updateDialogTitle(snap updater.Snapshot) string {
	switch snap.State {
	case updater.StateDownloading:
		return a.t("updateTitleDownloading")
	case updater.StateVerifying:
		return a.t("updateTitleVerifying")
	case updater.StateReadyToRestart:
		return a.t("updateTitleReady")
	case updater.StateError:
		return a.t("updateTitleError")
	default:
		return a.t("updateTitleAvailable")
	}
}

func (a *app) openUpdateReleasePage(c *ui.Context, owner toastOwner, url string) {
	if strings.TrimSpace(url) == "" {
		url = updater.DefaultReleasesURL
	}
	c.OpenURLThen(url, func(err error) {
		if err != nil {
			a.queueFor(owner, "error", a.tf("updateOpenReleasePageFailed", err.Error()), "")
			a.invalidate()
		}
	})
}

func (a *app) updateDialog(c *ui.Context, owner toastOwner) {
	ctrl := a.updateCtrl()
	if ctrl == nil || !ctrl.dialogOpen || ctrl.dialogOwner != owner {
		return
	}
	snap := a.updateSnapshot()
	tk := theme.TokensOf(c)

	workbenchModal(c.Key("workbench-update-dialog"), &ctrl.dialogOpen, func() {
		ui.Column(c).Width(480).
			Background(tk.BgElevated).Radius(12).Border(1, tk.Border).Clip().
			Children(func() {
				// Header
				ui.Row(c).FillWidth().Padding(16, 20, 12, 20).AlignItems(ui.Center).Gap(10).
					BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
					headIcon := lucide.ArrowUpCircle()
					headColor := tk.Blue
					if snap.State == updater.StateReadyToRestart {
						headIcon = lucide.CheckCircle()
						headColor = tk.SuccessText
					} else if snap.State == updater.StateError {
						headIcon = lucide.AlertCircle()
						headColor = tk.ErrorText
					}
					ui.Icon(c, headIcon).FontSize(scaledFont(18)).TextColor(headColor).Shrink(0)
					display.Text(c, a.updateDialogTitle(snap), display.TextProps{Strong: true, FontSize: scaledFont(15)}).
						Grow(1).Basis(0).MinWidth(0).SingleLine()
					if workbenchIconButton(c.Key("update-dialog-close"), lucide.X(), a.t("close"), buttonNormal, false).Clicked() {
						a.closeUpdateDialog()
					}
				})

				// Body
				ui.Column(c).FillWidth().Padding(16, 20, 16, 20).Gap(12).Children(func() {
					if snap.Release.Version != "" {
						currentLabel := "v" + snap.CurrentVersion
						nextLabel := "v" + snap.Release.Version
						display.Text(c, a.tf("updateVersionTransition", currentLabel, nextLabel), display.TextProps{
							Strong: true, FontSize: scaledFont(13),
						})
						if snap.State == updater.StateReadyToRestart {
							display.Text(c, a.tf("updateReadySummary", nextLabel), display.TextProps{
								FontSize: scaledFont(12),
							})
						}
						meta := make([]string, 0, 2)
						if snap.Release.PubDate != "" {
							meta = append(meta, a.tf("updatePublishedAt", formatUpdateDate(snap.Release.PubDate)))
						}
						if snap.Release.Asset.Size > 0 {
							meta = append(meta, a.tf("updatePackageSize", formatByteSize(snap.Release.Asset.Size)))
						}
						if len(meta) > 0 {
							display.Text(c, strings.Join(meta, " · "), display.TextProps{
								Type: display.TextSecondary, FontSize: scaledFont(11),
							})
						}
					}

					// Progress bar during download / verify
					if snap.State == updater.StateDownloading || snap.State == updater.StateVerifying {
						a.updateProgressBlock(c, snap)
					}

					// Error box
					if snap.State == updater.StateError && snap.Error != "" {
						ui.Column(c).FillWidth().Padding(10, 12).Radius(8).
							Background(tk.FillQuaternary).Border(1, tk.ErrorBorder).Children(func() {
							display.Text(c, snap.Error, display.TextProps{FontSize: scaledFont(11.5)}).TextColor(tk.ErrorText)
						})
					}

					// Release notes box
					if snap.Release.Version != "" {
						display.Text(c, a.t("updateReleaseNotes"), display.TextProps{
							Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11),
						})
						notes := snap.Release.NotesFor(string(a.locale))
						if notes == "" {
							notes = a.t("updateNoReleaseNotes")
						}
						ui.Box(c).FillWidth().Height(132).Radius(8).
							Background(tk.BgContainer).Border(1, tk.BorderSecondary).Clip().
							Children(func() {
								layout.ScrollArea(c, layout.ScrollAreaProps{State: &ctrl.notesScroll}, func() {
									ui.Column(c).FillWidth().Padding(10, 12).Selectable().Children(func() {
										display.Text(c, notes, display.TextProps{FontSize: scaledFont(12)})
									})
								}).Fill()
							})
					}

					// Ineligibility guidance when auto-install is unavailable
					if snap.State == updater.StateAvailable && !snap.CanAutoInstall() {
						if reason := a.updateIneligibleMessage(snap); reason != "" {
							display.Text(c, reason, display.TextProps{
								Type: display.TextSecondary, FontSize: scaledFont(11),
							})
						}
					}

					// Auto-download preference checkbox directly inside the update prompt
					autoDownload := a.autoDownloadUpdates
					wasAutoDownload := autoDownload
					choice.Checkbox(c.Key("update-dialog-auto-download"), &autoDownload, a.t("autoDownloadUpdates"), choice.CheckboxProps{Size: 14})
					if autoDownload != wasAutoDownload {
						a.setAutoDownloadPreference(autoDownload, owner)
					}
				})

				// Footer actions
				ui.Row(c).FillWidth().Padding(12, 20, 14, 20).AlignItems(ui.Center).Gap(8).
					Background(tk.BgContainerSecondary).BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).
					Children(func() {
						if snap.Release.Version != "" && (snap.State == updater.StateAvailable || snap.State == updater.StateReadyToRestart || snap.State == updater.StateError) {
							if workbenchButton(c.Key("update-skip"), a.t("updateSkipVersion"), buttonNormal, false).Clicked() {
								a.skipAvailableUpdate(owner)
							}
						}
						ui.Box(c).Grow(1)
						switch snap.State {
						case updater.StateDownloading:
							if workbenchButton(c.Key("update-cancel-dl"), a.t("updateCancelDownload"), buttonNormal, false).Clicked() {
								ctrl.mgr.Cancel()
								a.invalidate()
							}
							if workbenchButton(c.Key("update-bg-dl"), a.t("updateContinueInBackground"), buttonPrimary, false).Clicked() {
								a.closeUpdateDialog()
							}
						case updater.StateVerifying:
							if workbenchButton(c.Key("update-bg-verify"), a.t("updateContinueInBackground"), buttonPrimary, false).Clicked() {
								a.closeUpdateDialog()
							}
						case updater.StateReadyToRestart:
							if workbenchButton(c.Key("update-later-ready"), a.t("updateInstallLater"), buttonNormal, false).Clicked() {
								a.closeUpdateDialog()
							}
							if workbenchButton(c.Key("update-restart-now"), a.t("updateRestartAndInstall"), buttonPrimary, false).Clicked() {
								a.applyPreparedUpdate(owner)
							}
						case updater.StateError:
							if workbenchButton(c.Key("update-open-page-err"), a.t("updateOpenReleasePage"), buttonNormal, false).Clicked() {
								a.openUpdateReleasePage(c, owner, snap.Release.NotesURL)
							}
							if snap.Release.Version != "" && snap.CanAutoInstall() {
								if workbenchButton(c.Key("update-retry-dl"), a.t("updateRetry"), buttonPrimary, false).Clicked() {
									a.startUpdateDownload()
								}
							} else {
								if workbenchButton(c.Key("update-retry-check"), a.t("updateRetry"), buttonPrimary, false).Clicked() {
									a.closeUpdateDialog()
									a.checkForUpdates(owner)
								}
							}
						default: // StateAvailable
							if workbenchButton(c.Key("update-later"), a.t("updateLater"), buttonNormal, false).Clicked() {
								a.closeUpdateDialog()
							}
							if snap.CanAutoInstall() {
								if workbenchButton(c.Key("update-download-now"), a.t("updateDownloadAndInstall"), buttonPrimary, false).Clicked() {
									a.startUpdateDownload()
								}
							} else {
								if workbenchButton(c.Key("update-open-page"), a.t("updateOpenReleasePage"), buttonPrimary, false).Clicked() {
									a.openUpdateReleasePage(c, owner, snap.Release.NotesURL)
								}
							}
						}
					})
			})
	})
}

func (a *app) updateProgressBlock(c *ui.Context, snap updater.Snapshot) {
	tk := theme.TokensOf(c)
	var ratio float32
	var label string
	ver := "v" + snap.Release.Version
	if snap.State == updater.StateVerifying {
		ratio = 1
		label = a.tf("updateStatusVerifying", ver)
	} else if snap.TotalBytes > 0 {
		pct := int(min(100, (snap.DownloadedBytes*100)/snap.TotalBytes))
		ratio = float32(pct) / 100
		label = fmt.Sprintf("%s (%s / %s)", a.tf("updateStatusDownloading", ver, pct), formatByteSize(snap.DownloadedBytes), formatByteSize(snap.TotalBytes))
	} else {
		ratio = 0.25
		label = a.tf("updateStatusDownloadingBare", ver)
	}
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11.5)})
		bar := ui.Box(c).FillWidth().Height(6).Radius(3).Background(tk.FillTertiary).Clip()
		barW := bar.Bounds().W
		bar.Children(func() {
			fillW := barW * ratio
			if fillW < 6 && ratio > 0 {
				fillW = 6
			}
			ui.Box(c).Width(fillW).Height(6).Radius(3).Background(tk.Primary)
		})
	})
}

func (a *app) updatesCard(c *ui.Context) {
	a.settingsCard(c, a.t("softwareUpdates"), func() {
		wasCheck := a.autoCheckUpdates
		wasDownload := a.autoDownloadUpdates
		settingsCheckbox(c.Key("settings-auto-check-updates"), &a.autoCheckUpdates, a.t("autoCheckUpdates"), a.t("autoCheckUpdatesHelp"), false)
		downloadHelp := a.t("autoDownloadUpdatesHelp")
		if !a.autoCheckUpdates {
			downloadHelp = a.t("autoDownloadNeedsAutoCheck")
		}
		settingsCheckbox(c.Key("settings-auto-download-updates"), &a.autoDownloadUpdates, a.t("autoDownloadUpdates"), downloadHelp, !a.autoCheckUpdates)

		if a.autoCheckUpdates != wasCheck || a.autoDownloadUpdates != wasDownload {
			a.syncUpdaterPreferences()
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}

		a.settingsUpdateStatus(c, a.updateSnapshot())
	})
}

// settingsUpdateStatus renders a snapshot without changing the updater state.
func (a *app) settingsUpdateStatus(c *ui.Context, snap updater.Snapshot) {
	statusText := a.updateSettingsStatusText(snap)
	ui.Column(c).FillWidth().Padding(12).Gap(4).Children(func() {
		display.Text(c, statusText, display.TextProps{FontSize: scaledFont(12)})
		if !snap.LastCheckedAt.IsZero() {
			display.Text(c, a.tf("updateLastChecked", snap.LastCheckedAt.Local().Format("2006-01-02 15:04")), display.TextProps{
				Type: display.TextSecondary, FontSize: scaledFont(11),
			})
		}
	})

	ui.Row(c).FillWidth().Padding(0, 12, 12).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
		switch snap.State {
		case updater.StateChecking:
			workbenchButton(c.Key("settings-update-checking"), a.t("updateStatusChecking"), buttonNormal, true)
		case updater.StateAvailable:
			if workbenchButton(c.Key("settings-update-view"), a.t("updateViewDetailsAction"), buttonPrimary, false).Clicked() {
				a.openUpdateDialog(toastSettings)
			}
			if workbenchButton(c.Key("settings-update-recheck"), a.t("checkForUpdates"), buttonNormal, false).Clicked() {
				a.checkForUpdates(toastSettings)
			}
		case updater.StateDownloading, updater.StateVerifying:
			if workbenchButton(c.Key("settings-update-progress"), a.t("updateViewDetailsAction"), buttonPrimary, false).Clicked() {
				a.openUpdateDialog(toastSettings)
			}
			if snap.State == updater.StateDownloading {
				if workbenchButton(c.Key("settings-update-cancel"), a.t("updateCancelDownload"), buttonNormal, false).Clicked() {
					if ctrl := a.updateCtrl(); ctrl != nil && ctrl.mgr != nil {
						ctrl.mgr.Cancel()
						a.invalidate()
					}
				}
			}
		case updater.StateReadyToRestart:
			if workbenchButton(c.Key("settings-update-restart"), a.t("updateRestartAndInstall"), buttonPrimary, false).Clicked() {
				a.applyPreparedUpdate(toastSettings)
			}
			if workbenchButton(c.Key("settings-update-details"), a.t("updateViewDetailsAction"), buttonNormal, false).Clicked() {
				a.openUpdateDialog(toastSettings)
			}
		default:
			if workbenchButton(c.Key("settings-check-updates"), a.t("checkForUpdates"), buttonNormal, false).Clicked() {
				a.checkForUpdates(toastSettings)
			}
		}
	})
}

func (a *app) updateSettingsStatusText(snap updater.Snapshot) string {
	ver := "v" + snap.Release.Version
	switch snap.State {
	case updater.StateChecking:
		return a.t("updateStatusChecking")
	case updater.StateUpToDate:
		return a.t("updateStatusUpToDate")
	case updater.StateAvailable:
		return a.tf("updateStatusAvailable", ver)
	case updater.StateDownloading:
		if snap.TotalBytes > 0 {
			pct := int(min(100, (snap.DownloadedBytes*100)/snap.TotalBytes))
			return a.tf("updateStatusDownloading", ver, pct)
		}
		return a.tf("updateStatusDownloadingBare", ver)
	case updater.StateVerifying:
		return a.tf("updateStatusVerifying", ver)
	case updater.StateReadyToRestart:
		return a.tf("updateStatusReady", ver)
	case updater.StateError:
		return a.tf("updateStatusError", snap.Error)
	default:
		return a.t("updateStatusIdle")
	}
}

func formatUpdateDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Local().Format("2006-01-02")
	}
	return raw
}

func formatByteSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(suffixes) {
		exp = len(suffixes) - 1
	}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), suffixes[exp])
}
