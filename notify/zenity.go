package notify

import (
	"fmt"
	"os/exec"
)

type Zenity struct {
}

func NewZenity() Notifier {

	return &Zenity{}
}

func (s *Zenity) Send(summary, message string) {

	// zenity --info --text='<span foreground="blue" size="x-large">Blue text</span> is <i>cool</i>!' --icon-name=appointment-soon --title="Reminder"
	msg := fmt.Sprintf("<span foreground=\"blue\" size=\"large\">%s</span>\n<span>%s</span>", summary, message)
	exec.Command("zenity", "--info", "--icon-name", "appointment-soon", "--title", "Reminder", "--text", msg).Run()
}
