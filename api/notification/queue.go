// BLOCKED: this file depends on github.com/pennsieve/email-service/client, which
// is not yet tagged/published (see ../go.mod). It will not compile until that
// module is available and added to go.mod. The logic and call-site swap are
// complete and ready for review; only the dependency wiring is pending.

package notification

import (
	"context"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	emailclient "github.com/pennsieve/email-service/client"
	log "github.com/sirupsen/logrus"
)

// QueueNotifier is a Notifier that sends proposal emails through the Pennsieve
// email-service by enqueuing requests on its SQS queue, instead of rendering a
// template and calling SES directly. The email-service consumer renders the
// template (from the shared email-templates) and delivers via SES.
//
// It is a drop-in replacement for EmailNotifier behind the Notifier interface,
// so the call sites in api/service do not change.
type QueueNotifier struct {
	ctx    context.Context
	client *emailclient.Client
}

// NewQueueNotifier constructs a QueueNotifier. EMAIL_SERVICE_QUEUE_URL is the
// URL of the email-service send queue for the environment.
func NewQueueNotifier(ctx context.Context) (*QueueNotifier, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config for QueueNotifier: %w", err)
	}
	queueURL := os.Getenv("EMAIL_SERVICE_QUEUE_URL")
	if queueURL == "" {
		return nil, fmt.Errorf("EMAIL_SERVICE_QUEUE_URL is not set")
	}
	return &QueueNotifier{
		ctx:    ctx,
		client: emailclient.New(sqs.NewFromConfig(cfg), queueURL),
	}, nil
}

// send enqueues one request per recipient. The email-service handles a "to" of
// one recipient per message (it dedupes/journals per recipient), so we fan out
// here, matching the previous SES-per-recipient behavior.
func (q *QueueNotifier) send(build func(to emailclient.To) emailclient.EmailRequest, recipients []string) error {
	for _, addr := range recipients {
		if addr == "" {
			continue
		}
		req := build(emailclient.To{Email: addr})
		if err := q.client.Send(q.ctx, req); err != nil {
			log.WithFields(log.Fields{"recipient": addr, "messageId": req.MessageId, "error": fmt.Sprintf("%+v", err)}).
				Error("QueueNotifier.send()")
			return err
		}
	}
	return nil
}

func (q *QueueNotifier) ProposalSubmitted(a MessageAttributes, recipients []string) error {
	return q.send(func(to emailclient.To) emailclient.EmailRequest {
		return emailclient.DatasetProposalSubmitted(to, emailclient.DatasetProposalSubmittedArgs{
			AppURL:          a["AppURL"],
			AuthorEmail:     a["AuthorEmail"],
			AuthorName:      a["AuthorName"],
			ProposalTitle:   a["ProposalTitle"],
			WorkspaceName:   a["WorkspaceName"],
			WorkspaceNodeId: a["WorkspaceNodeId"],
		})
	}, recipients)
}

func (q *QueueNotifier) ProposalWithdrawn(a MessageAttributes, recipients []string) error {
	return q.send(func(to emailclient.To) emailclient.EmailRequest {
		return emailclient.DatasetProposalWithdrawn(to, emailclient.DatasetProposalWithdrawnArgs{
			AppURL:          a["AppURL"],
			AuthorEmail:     a["AuthorEmail"],
			AuthorName:      a["AuthorName"],
			ProposalTitle:   a["ProposalTitle"],
			WorkspaceName:   a["WorkspaceName"],
			WorkspaceNodeId: a["WorkspaceNodeId"],
		})
	}, recipients)
}

func (q *QueueNotifier) ProposalAccepted(a MessageAttributes, recipients []string) error {
	return q.send(func(to emailclient.To) emailclient.EmailRequest {
		return emailclient.DatasetProposalAccepted(to, emailclient.DatasetProposalAcceptedArgs{
			AppURL:                 a["AppURL"],
			ProposalTitle:          a["ProposalTitle"],
			WelcomeWorkspaceNodeId: a["WelcomeWorkspaceNodeId"],
			WorkspaceName:          a["WorkspaceName"],
		})
	}, recipients)
}

func (q *QueueNotifier) ProposalRejected(a MessageAttributes, recipients []string) error {
	return q.send(func(to emailclient.To) emailclient.EmailRequest {
		return emailclient.DatasetProposalRejected(to, emailclient.DatasetProposalRejectedArgs{
			AppURL:                 a["AppURL"],
			ProposalTitle:          a["ProposalTitle"],
			WelcomeWorkspaceNodeId: a["WelcomeWorkspaceNodeId"],
			WorkspaceName:          a["WorkspaceName"],
		})
	}, recipients)
}
