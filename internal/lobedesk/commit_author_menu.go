package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
)

type commitAuthorIdentity struct{ Name, Email string }

// Keep name and email from the same author on which the menu opened, even if
// history refreshes while it remains open. No repository work is involved.
func (a *app) commitAuthorMenu(c *ui.Context, author ui.Element, name, email string) {
	open := ui.Local(author, "contextMenu", func() bool { return false })
	captured := ui.Local(author, "copy-author-identity", func() commitAuthorIdentity { return commitAuthorIdentity{} })
	if !*open {
		*captured = commitAuthorIdentity{Name: name, Email: email}
	}
	switch overlay.ContextMenu(c, author, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy-author", Label: a.t("copyAuthor"), Icon: lucide.Copy(), Disabled: captured.Email == ""},
		{Key: "copy-author-name", Label: a.t("copyAuthorName"), Icon: lucide.Copy(), Disabled: captured.Name == ""},
		{Key: "copy-author-email", Label: a.t("copyAuthorEmail"), Icon: lucide.Copy(), Disabled: captured.Email == ""},
	}}) {
	case "copy-author":
		a.copyText(captured.Name + " <" + captured.Email + ">")
	case "copy-author-name":
		a.copyText(captured.Name)
	case "copy-author-email":
		a.copyText(captured.Email)
	}
}
