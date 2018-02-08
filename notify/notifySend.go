package notify

import "os/exec"

type NotifySend struct {
	icon string
}

func NewNotifySend(icon string) Notifier {

	return &NotifySend{
		icon: icon,
	}
}

func (s *NotifySend) Send(summary, message string) {

	exec.Command("notify-send", "-i", s.icon, summary, message).Run()
}
