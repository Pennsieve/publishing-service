package notification

import (
	"context"
	"fmt"
	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/aws/ses"
	sesTypes "github.com/pennsieve/publishing-service/api/aws/ses/types"
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
		log.WithFields(log.Fields{"key": key, "search": search, "replace": replace}).Info("EmailNotifier.replaceTemplateFields()")
		modified = strings.Replace(modified, search, replace, -1)
	}

	return modified
}

func (e *EmailNotifier) generateAndSendEmail(s3Bucket string, s3Key string, messageAttributes MessageAttributes, recipients []string, subject string) error {
	log.WithFields(log.Fields{}).Info("EmailNotifier.generateAndSendEmail()")

	// load email template
	template, err := e.fileReader.ReadFile(e.ctx, s3Bucket, s3Key)
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("EmailNotifier.generateAndSendEmail()")
		return err
	}

	// substitute values
	body := e.replaceTemplateFields(template, messageAttributes)

	// send email
	err = e.emailAgent.SendMessage(e.ctx, e.sender, recipients, subject, body, sesTypes.HTML)

	return err
}

func (e *EmailNotifier) ProposalSubmitted(messageAttributes MessageAttributes, recipients []string) error {
	subject := "A Dataset Proposal has been submitted"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_SUBMITTED")
	log.WithFields(log.Fields{
		"messageAttributes": fmt.Sprintf("%s", messageAttributes),
		"subject":           subject,
		"s3Bucket":          s3Bucket,
		"s3Key":             s3Key,
		"recipients":        recipients}).Info("EmailNotifier.ProposalSubmitted()")

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}

func (e *EmailNotifier) ProposalWithdrawn(messageAttributes MessageAttributes, recipients []string) error {
	subject := "A Dataset Proposal has been withdrawn"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_WITHDRAWN")
	log.WithFields(log.Fields{
		"messageAttributes": fmt.Sprintf("%s", messageAttributes),
		"subject":           subject,
		"s3Bucket":          s3Bucket,
		"s3Key":             s3Key,
		"recipients":        recipients}).Info("EmailNotifier.ProposalWithdrawn()")

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}

func (e *EmailNotifier) ProposalAccepted(messageAttributes MessageAttributes, recipients []string) error {
	subject := "Your Dataset Proposal has been accepted"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_ACCEPTED")
	log.WithFields(log.Fields{
		"messageAttributes": fmt.Sprintf("%s", messageAttributes),
		"subject":           subject,
		"s3Bucket":          s3Bucket,
		"s3Key":             s3Key,
		"recipients":        recipients}).Info("EmailNotifier.ProposalAccepted()")

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}

func (e *EmailNotifier) ProposalRejected(messageAttributes MessageAttributes, recipients []string) error {
	subject := "Your Dataset Proposal has been rejected"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_REJECTED")
	log.WithFields(log.Fields{
		"messageAttributes": fmt.Sprintf("%s", messageAttributes),
		"subject":           subject,
		"s3Bucket":          s3Bucket,
		"s3Key":             s3Key,
		"recipients":        recipients}).Info("EmailNotifier.ProposalRejected()")

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}
