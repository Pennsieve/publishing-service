package ses

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	sesTypes "github.com/pennsieve/publishing-service/api/aws/ses/types"
	log "github.com/sirupsen/logrus"
)

func MakeEmailer() *Emailer {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		// TODO: handle error
	}

	client := ses.NewFromConfig(cfg)
	return &Emailer{
		Client:  client,
		CharSet: "UTF-8",
	}
}

type Emailer struct {
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
		log.WithFields(log.Fields{"SendMessage": "failure", "error": fmt.Sprintf("%+v", err)}).Info("Emailer.SendMessage()")
		return err
	}

	log.WithFields(log.Fields{"SendMessage": "success", "result": fmt.Sprintf("%+v", result)}).Info("Emailer.SendMessage()")
	return nil
}
