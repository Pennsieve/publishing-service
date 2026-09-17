package notification

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/aws/ses"
	sesTypes "github.com/pennsieve/publishing-service/api/aws/ses/types"
	"github.com/pennsieve/publishing-service/api/logging"
)

// NewEmailNotifier builds a Notifier that renders a template from S3 and sends
// it directly via SES. Superseded by NewQueueNotifier (the email-service queue
// path); retained until the direct-SES path is removed. logger is the
// request-scoped logger built at the entrypoint.
func NewEmailNotifier(ctx context.Context, logger *slog.Logger) *EmailNotifier {
	return &EmailNotifier{
		ctx:        ctx,
		logger:     logger,
		sender:     fmt.Sprintf("support@%s", os.Getenv("PENNSIEVE_DOMAIN")),
		emailAgent: ses.MakeEmailer(logger),
		fileReader: s3.MakeFileReader(logger),
	}
}

type EmailNotifier struct {
	ctx        context.Context
	logger     *slog.Logger
	sender     string
	fileReader *s3.FileReader
	emailAgent *ses.Emailer
}

func (e *EmailNotifier) replaceTemplateFields(template string, messageAttributes MessageAttributes) string {
	modified := template

	for key := range messageAttributes {
		search := fmt.Sprintf("${%s}", key)
		replace := messageAttributes[key]
		// Only the key is logged: the values are proposal/author details.
		e.logger.Debug("substituting email template field", slog.String("field", key))
		modified = strings.Replace(modified, search, replace, -1)
	}

	return modified
}

func (e *EmailNotifier) generateAndSendEmail(s3Bucket string, s3Key string, messageAttributes MessageAttributes, recipients []string, subject string) error {
	// load email template
	template, err := e.fileReader.ReadFile(e.ctx, s3Bucket, s3Key)
	if err != nil {
		e.logger.Error("failed to read the email template from S3",
			slog.String(logging.KeyS3Bucket, s3Bucket),
			slog.String(logging.KeyS3Key, s3Key),
			slog.Any(logging.KeyError, err))
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
	// Recipient addresses and the message attributes (author name/email) are
	// deliberately not logged; only their count and the template used.
	e.logger.Info("sending proposal email",
		slog.String(logging.KeyAction, "submitted"),
		slog.String(logging.KeySubject, subject),
		slog.String(logging.KeyTemplate, s3Key),
		slog.Int(logging.KeyCount, len(recipients)))

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}

func (e *EmailNotifier) ProposalWithdrawn(messageAttributes MessageAttributes, recipients []string) error {
	subject := "A Dataset Proposal has been withdrawn"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_WITHDRAWN")
	// Recipient addresses and the message attributes (author name/email) are
	// deliberately not logged; only their count and the template used.
	e.logger.Info("sending proposal email",
		slog.String(logging.KeyAction, "withdrawn"),
		slog.String(logging.KeySubject, subject),
		slog.String(logging.KeyTemplate, s3Key),
		slog.Int(logging.KeyCount, len(recipients)))

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}

func (e *EmailNotifier) ProposalAccepted(messageAttributes MessageAttributes, recipients []string) error {
	subject := "Your Dataset Proposal has been accepted"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_ACCEPTED")
	// Recipient addresses and the message attributes (author name/email) are
	// deliberately not logged; only their count and the template used.
	e.logger.Info("sending proposal email",
		slog.String(logging.KeyAction, "accepted"),
		slog.String(logging.KeySubject, subject),
		slog.String(logging.KeyTemplate, s3Key),
		slog.Int(logging.KeyCount, len(recipients)))

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}

func (e *EmailNotifier) ProposalRejected(messageAttributes MessageAttributes, recipients []string) error {
	subject := "Your Dataset Proposal has been rejected"
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_REJECTED")
	// Recipient addresses and the message attributes (author name/email) are
	// deliberately not logged; only their count and the template used.
	e.logger.Info("sending proposal email",
		slog.String(logging.KeyAction, "rejected"),
		slog.String(logging.KeySubject, subject),
		slog.String(logging.KeyTemplate, s3Key),
		slog.Int(logging.KeyCount, len(recipients)))

	return e.generateAndSendEmail(s3Bucket, s3Key, messageAttributes, recipients, subject)
}
