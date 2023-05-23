package notification

import "fmt"

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
}
