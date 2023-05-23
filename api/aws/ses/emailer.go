package ses

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
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

// TODO: make `to` an array of recipients
func (emailer *Emailer) SendMessage(ctx context.Context, sender string, recipients []string, subject string, body string) error {
	// compose email message
	message := &ses.SendEmailInput{
		Destination: &types.Destination{
			BccAddresses: nil,
			CcAddresses:  nil,
			ToAddresses:  recipients,
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: nil,
				Text: &types.Content{
					Data:    &body,
					Charset: &emailer.CharSet,
				},
			},
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
