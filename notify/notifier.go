package notify

type Notifier interface {
	Send(summary, message string)
}
