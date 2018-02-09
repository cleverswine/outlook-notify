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

	msg := fmt.Sprintf("<span foreground=\"blue\" size=\"large\">%s</span>\n<span>%s</span>", summary, message)
	exec.Command("zenity", "--info", "--icon-name", "appointment-soon", "--title", "Reminder", "--text", msg).Run()
}
