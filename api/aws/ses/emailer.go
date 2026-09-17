package ses

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	sesTypes "github.com/pennsieve/publishing-service/api/aws/ses/types"
	"github.com/pennsieve/publishing-service/api/logging"
)

// MakeEmailer builds an Emailer. logger is the request-scoped logger built at
// the entrypoint, held on the struct so no method has to reach for
// slog.Default.
func MakeEmailer(logger *slog.Logger) *Emailer {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("config.LoadDefaultConfig() failed building emailer", slog.Any(logging.KeyError, err))
	}

	client := ses.NewFromConfig(cfg)
	return &Emailer{
		logger:  logger,
		Client:  client,
		CharSet: "UTF-8",
	}
}

type Emailer struct {
	logger  *slog.Logger
	Client  *ses.Client
	CharSet string
}

func (emailer *Emailer) messageBody(body string, format sesTypes.MessageFormat) *types.Body {
	switch format {
	case sesTypes.HTML:
		return &types.Body{
			Html: &types.Content{
				Data:    &body,
				Charset: &emailer.CharSet,
			},
			Text: nil,
		}
	case sesTypes.Text:
		return &types.Body{
			Html: nil,
			Text: &types.Content{
				Data:    &body,
				Charset: &emailer.CharSet,
			},
		}
	}
	return nil
}

func (emailer *Emailer) SendMessage(ctx context.Context, sender string, recipients []string, subject string, body string, format sesTypes.MessageFormat) error {
	// compose email message
	message := &ses.SendEmailInput{
		Destination: &types.Destination{
			BccAddresses: nil,
			CcAddresses:  nil,
			ToAddresses:  recipients,
		},
		Message: &types.Message{
			Body: emailer.messageBody(body, format),
			Subject: &types.Content{
				Data:    &subject,
				Charset: &emailer.CharSet,
			},
		},
		Source:               &sender,
		ConfigurationSetName: nil,
		ReplyToAddresses:     nil,
		ReturnPath:           nil,
		ReturnPathArn:        nil,
		SourceArn:            nil,
		Tags:                 nil,
	}

	result, err := emailer.Client.SendEmail(ctx, message)
	if err != nil {
		emailer.logger.Error("ses SendEmail() failed",
			slog.Int(logging.KeyCount, len(recipients)),
			slog.String(logging.KeySubject, subject),
			slog.Any(logging.KeyError, err))
		return err
	}

	emailer.logger.Info("sent email via SES",
		slog.Int(logging.KeyCount, len(recipients)),
		slog.String(logging.KeySubject, subject),
		slog.String(logging.KeyMessageID, aws.ToString(result.MessageId)))
	return nil
}
