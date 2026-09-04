package notification

import "fmt"

type Notification int64

const (
	Submitted Notification = iota
	Withdrawn
	Accepted
	Rejected
)

// String renders the notification kind for logs, so log lines carry
// "submitted" rather than the bare enum ordinal.
func (n Notification) String() string {
	switch n {
	case Submitted:
		return "submitted"
	case Withdrawn:
		return "withdrawn"
	case Accepted:
		return "accepted"
	case Rejected:
		return "rejected"
	default:
		return "unknown"
	}
}

type MessageAttributes map[string]string

func (ma MessageAttributes) String() string {
	result := ""
	sep := "|_|"
	for key := range ma {
		result = fmt.Sprintf("%s%s=%s%s", result, key, ma[key], sep)
	}
	return result
}

type Notifier interface {
	ProposalSubmitted(messageAttributes MessageAttributes, recipients []string) error
	ProposalWithdrawn(messageAttributes MessageAttributes, recipients []string) error
	ProposalAccepted(messageAttributes MessageAttributes, recipients []string) error
	ProposalRejected(messageAttributes MessageAttributes, recipients []string) error
}
