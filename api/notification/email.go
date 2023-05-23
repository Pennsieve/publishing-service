package notification

import (
	"context"
	"fmt"
	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/aws/ses"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func NewEmailNotifier(ctx context.Context) *EmailNotifier {
	return &EmailNotifier{
		ctx:        ctx,
		sender:     fmt.Sprintf("support@%s", os.Getenv("PENNSIEVE_DOMAIN")),
		emailAgent: ses.MakeEmailer(),
		fileReader: s3.MakeFileReader(),
	}
}

type EmailNotifier struct {
	ctx        context.Context
	sender     string
	fileReader *s3.FileReader
	emailAgent *ses.Emailer
}

func (e *EmailNotifier) replaceTemplateFields(template string, messageAttributes MessageAttributes) string {
	modified := template

	for key := range messageAttributes {
		search := fmt.Sprintf("${%s}", key)
		replace := messageAttributes[key]
		log.WithFields(log.Fields{"key": key, "search": search, "replace": replace}).Info("replaceTemplateFields()")
		modified = strings.Replace(modified, search, replace, -1)
	}

	return modified
}

func (e *EmailNotifier) ProposalSubmitted(messageAttributes MessageAttributes, recipients []string) error {
	log.WithFields(log.Fields{"messageAttributes": fmt.Sprintf("%s", messageAttributes), "recipients": recipients}).Info("EmailNotifier.ProposalSubmitted()")

	var err error
	subject := "A Dataset Proposal has been submitted"

	// load template
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_SUBMITTED")
	log.WithFields(log.Fields{"s3Bucket": s3Bucket, "s3Key": s3Key}).Info("EmailNotifier.ProposalSubmitted()")

	template, err := e.fileReader.ReadFile(e.ctx, s3Bucket, s3Key)
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("service.loadEmailTemplate()")
		return err
	}

	// substitute values
	body := e.replaceTemplateFields(template, messageAttributes)

	// send email
	err = e.emailAgent.SendMessage(e.ctx, e.sender, recipients, subject, body)

	return err
}
