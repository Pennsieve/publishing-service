package service

import (
	"context"
	"fmt"
	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/models"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

type MessageAttributes map[string]string

type EmailMessage struct {
	Subject string
	Body    string
}

func loadEmailTemplate(ctx context.Context, s3Bucket string, s3Key string) (*string, error) {
	log.WithFields(log.Fields{"s3Bucket": s3Bucket, "s3Key": s3Key}).Info("service.loadEmailTemplate()")
	reader := s3.MakeFileReader()
	source, err := reader.ReadFile(ctx, s3Bucket, s3Key)
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("service.loadEmailTemplate()")
		return nil, err
	}
	log.WithFields(log.Fields{"source": source}).Info("service.loadEmailTemplate()")
	return source, nil
}

func replaceTemplateFields(template *string, messageAttributes MessageAttributes) string {
	modified := *template

	for key := range messageAttributes {
		search := fmt.Sprintf("${%s}", key)
		replace := messageAttributes[key]
		log.WithFields(log.Fields{"key": key, "search": search, "replace": replace}).Info("replaceTemplateFields()")
		modified = strings.Replace(modified, search, replace, -1)
	}

	return modified
}

func MakeMessageAttributes(proposal *models.DatasetProposal, repository *models.Repository) *MessageAttributes {
	return &MessageAttributes{
		"AppURL":          os.Getenv("PENNSIEVE_DOMAIN"),
		"AuthorName":      proposal.OwnerName,
		"AuthorEmail":     proposal.EmailAddress,
		"ProposalTitle":   proposal.Name,
		"WorkspaceName":   repository.Name,
		"WorkspaceNodeId": repository.OrganizationNodeId,
	}
}

func ProposalSubmittedMessage(ctx context.Context, messageAttributes *MessageAttributes) (*EmailMessage, error) {
	s3Bucket := os.Getenv("EMAIL_TEMPLATE_BUCKET")
	s3Key := os.Getenv("EMAIL_TEMPLATE_SUBMITTED")
	log.WithFields(log.Fields{"s3Bucket": s3Bucket, "s3Key": s3Key, "messageAttributes": fmt.Sprintf("%+v", messageAttributes)}).Info("service.ProposalSubmittedMessage()")

	// read template file
	template, err := loadEmailTemplate(ctx, s3Bucket, s3Key)
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("service.ProposalSubmittedMessage()")
		return nil, err
	}

	// substitute values
	modified := replaceTemplateFields(template, *messageAttributes)
	log.WithFields(log.Fields{"modified": modified}).Info("service.ProposalSubmittedMessage()")

	return &EmailMessage{
		Subject: "A Dataset Proposal has been Submitted",
		Body:    modified,
	}, nil
}
