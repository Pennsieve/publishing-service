package ses

import
import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/aws/awserr"
	log "github.com/sirupsen/logrus"
)
{
	"github.com/aws/aws-sdk-go/service/ses"
}

func MakeEmailer() *Emailer {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		// TODO: handle error
	}
	svc := ses.New(sess)
	return &Emailer{
		Mailer: svc,
		CharSet: "UTF-8",
	}
}

type Emailer struct {
	Mailer *ses.SES
	CharSet string
}

// TODO: make `to` an array of recipients
func (emailer *Emailer) SendMessage(from string, to string, subject string, body string) error {
	// compose email message
	message := &ses.SendEmailInput{
		ConfigurationSetName: nil,
		Destination:          &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{aws.String(to),},
			},
		Message:              &ses.Message{
			Body: &ses.Body{
				//Html: &ses.Content{
				//	Charset: aws.String(CharSet),
				//	Data:    aws.String(HtmlBody),
				//},
				Text: &ses.Content{
					Charset: aws.String(emailer.CharSet),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(emailer.CharSet),
				Data:    aws.String(subject),
			},
		},
		ReplyToAddresses:     nil,
		ReturnPath:           nil,
		ReturnPathArn:        nil,
		Source:               aws.String(from),
		SourceArn:            nil,
		Tags:                 nil,
	}

	result, err := emailer.Mailer.SendEmail(message)
	if err != nil {
		if awsError, ok := err.(awserr.Error); ok {
			switch awsError.Code() {
			case ses.ErrCodeMessageRejected:
				log.WithFields(log.Fields{"SendMessage":"error", "error":"Rejected", "reason":fmt.Sprintf("%+v",awsError.Error())}).Error("emailer.SendMessage()")
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				log.WithFields(log.Fields{"SendMessage":"error", "error":"FromDomainNotVerified", "reason":fmt.Sprintf("%+v",awsError.Error())}).Error("emailer.SendMessage()")
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				log.WithFields(log.Fields{"SendMessage":"error", "error":"ConfigurationSetDoesNotExist", "reason":fmt.Sprintf("%+v",awsError.Error())}).Error("emailer.SendMessage()")
			default:
				log.WithFields(log.Fields{"SendMessage":"error", "error":"(unspecified)", "reason":fmt.Sprintf("%+v",awsError.Error())}).Error("emailer.SendMessage()")
			}
		} else {
			log.WithFields(log.Fields{"SendMessage":"error", "error":"(unspecified)", "reason":fmt.Sprintf("%+v",err.Error())}).Error("emailer.SendMessage()")
		}
	}

	log.WithFields(log.Fields{"SendMessage":"success", "result": fmt.Sprintf("%+v",result)}).Info("emailer.SendMessage()")

	return err
}
