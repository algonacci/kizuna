package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type desktop struct{ win *mygo.Window }

func (d *desktop) Update(fn func()) { d.win.Update(fn) }
func (d *desktop) PickFolder() (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: d.win, Title: "Open Git Repository", Directory: true, ShowHiddenFiles: true})
	if len(paths) == 0 {
		return "", err
	}
	return paths[0], err
}
func (d *desktop) OpenPath(path string) error { return mygo.Shell.OpenPath(path) }
func (d *desktop) PickCloneFolder(defaultPath string) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: d.win, Title: "Choose Parent Folder for Clone", Directory: true, DefaultPath: defaultPath, ShowHiddenFiles: true})
	if len(paths) == 0 {
		return "", err
	}
	return paths[0], err
}
func (d *desktop) OpenTerminal(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("/usr/bin/open", "-a", "Terminal", path).Run()
	case "windows":
		cmd := exec.Command("powershell.exe", "-NoExit")
		cmd.Dir = path
		if err := cmd.Start(); err != nil {
			return err
		}
		go cmd.Wait()
		return nil
	default:
		for _, name := range []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xterm"} {
			if binary, err := exec.LookPath(name); err == nil {
				cmd := exec.Command(binary)
				cmd.Dir = path
				if err = cmd.Start(); err != nil {
					return err
				}
				go cmd.Wait()
				return nil
			}
		}
		return fmt.Errorf("No supported terminal found")
	}
}
func (a *app) installMenu() {
	item := func(label, key string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: key, Click: func(*mygo.MenuItem, *mygo.Window) { a.host.Update(fn) }}
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			item("Open Repository…", "CmdOrCtrl+O", a.pickRepository),
			item("Clone Repository…", "", a.startClone),
			item("New Worktree…", "CmdOrCtrl+N", a.startCreate),
			item("Refresh", "CmdOrCtrl+R", a.refresh),
		}},
		{Label: "Repository", Submenu: []*mygo.MenuItem{
			item("Fetch", "", func() { a.remoteAction("fetch") }),
			item("Pull", "", func() { a.remoteAction("pull") }),
			item("Push", "", func() { a.remoteAction("push") }),
			item("Open Folder", "", func() { a.openSelected(false) }),
			item("Open Terminal", "", func() { a.openSelected(true) }),
		}},
		{Role: mygo.RoleEditMenu}, {Role: mygo.RoleWindowMenu},
		{Label: "Help", Submenu: []*mygo.MenuItem{item("Keyboard Shortcuts", "CmdOrCtrl+/", func() { a.keyboardHelp = true })}},
	}))
}
func main() {
	mygo.Theme.SetSource(mygo.ThemeDark)
	d := &desktop{}
	a := newApp(d)
	if dir, err := os.UserConfigDir(); err == nil {
		a.configPath = filepath.Join(dir, "kizuna", "recent.json")
		a.recent = readRecent(a.configPath)
	}
	mygo.App.WhenReady(func() {
		style := mygo.TitleBarDefault
		if runtime.GOOS == "darwin" {
			style = mygo.TitleBarHidden
		}
		d.win = mygo.NewWindow(mygo.WindowOptions{Title: "Kizuna", Width: 1180, Height: 780, MinWidth: 820, MinHeight: 600, TitleBarStyle: style, TitleBarHeight: 68, Content: ui.View(a.view)})
		a.installMenu()
		d.win.OnClosed(func() { a.cancel(); mygo.App.Quit() })
		d.win.OnFocus(func() { d.Update(a.refresh) })
		if len(os.Args) > 1 {
			a.openRepository(os.Args[1])
		} else if len(a.recent) > 0 {
			a.openRepository(a.recent[0])
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
