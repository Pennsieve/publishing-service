package service

import (
	"context"
	"fmt"
	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/models"
	"os"
	"strings"
)

type MessageAttributes map[string]string

type EmailMessage struct {
	Subject string
	Body    string
}

func loadEmailTemplate(ctx context.Context, s3Bucket string, s3Key string) (*string, error) {
	reader := s3.MakeFileReader()
	source, err := reader.ReadFile(ctx, os.Getenv(s3Bucket), os.Getenv(s3Key))
	if err != nil {
		// TODO: better error handling
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

func ProposalSubmittedMessage(ctx context.Context, messageAttributes MessageAttributes) (*EmailMessage, error) {
	// read template file
	template, err := loadEmailTemplate(ctx, "EMAIL_TEMPLATE_BUCKET", "EMAIL_TEMPLATE_SUBMITTED")
	if err != nil {
		// TODO: better error handling
		return nil, err
	}

	// substitute values
	modified := *template
	for key := range messageAttributes {
		search := fmt.Sprintf("${%s}", key)
		replace := messageAttributes[key]
		modified = strings.Replace(modified, search, replace, -1)
	}

	return &EmailMessage{
		Subject: "A Dataset Proposal has been Submitted",
		Body:    modified,
	}, nil
}
