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
	return source, nil
}

func GenerateMessageAttributes(proposal *models.DatasetProposal, repository *models.Repository) MessageAttributes {
	return MessageAttributes{
		"AppURL":          os.Getenv("PENNSIEVE_DOMAIN"),
		"AuthorName":      proposal.OwnerName,
		"AuthorEmail":     proposal.EmailAddress,
		"ProposalTitle":   proposal.Name,
		"WorkspaceName":   repository.Name,
		"WorkspaceNodeId": repository.OrganizationNodeId,
	}
}

func ProposalSubmittedMessage(ctx context.Context, messageAttributes MessageAttributes) (EmailMessage, error) {
	log.WithFields(log.Fields{"messageAttributes": fmt.Sprintf("%+v", messageAttributes)}).Info("service.ProposalSubmittedMessage()")

	// read template file
	template, err := loadEmailTemplate(ctx,
		os.Getenv("EMAIL_TEMPLATE_BUCKET"),
		os.Getenv("EMAIL_TEMPLATE_SUBMITTED"))
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("service.ProposalSubmittedMessage()")
		return EmailMessage{}, err
	}

	// substitute values
	modified := *template
	for key := range messageAttributes {
		search := fmt.Sprintf("${%s}", key)
		replace := messageAttributes[key]
		modified = strings.Replace(modified, search, replace, -1)
	}

	return EmailMessage{
		Subject: "A Dataset Proposal has been Submitted",
		Body:    modified,
	}, nil
}
