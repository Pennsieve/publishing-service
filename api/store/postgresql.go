package store

import (
	"context"
	"database/sql"
	"fmt"
	model "github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/pennsieve/publishing-service/api/models"
	log "github.com/sirupsen/logrus"
)

type PennsievePublishingStore interface {
	CreateDatasetForAcceptedProposal(ctx context.Context, proposal *models.DatasetProposal) (*CreatedDataset, error)
}

func NewPennsieveStore(db *sql.DB, orgId int64) *pennsieveStore {
	dbTx, err := db.BeginTx(context.TODO(), nil)
	if err != nil {
		panic(err)
	}

	return &pennsieveStore{
		orgId: orgId,
		db:    db,
		q:     pgdb.New(dbTx),
	}
}

type pennsieveStore struct {
	orgId int64
	db    *sql.DB
	q     *pgdb.Queries
}

type CreatedDataset struct {
	User         *model.User
	Organization *model.Organization
	Dataset      *model.Dataset
}

func (p *pennsieveStore) CreateDatasetForAcceptedProposal(ctx context.Context, proposal *models.DatasetProposal) (*CreatedDataset, error) {
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Info("pennsieveStore.CreateDatasetForAcceptedProposal()")

	var err error

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		log.WithFields(log.Fields{"failure": "BeginTx", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to begin database transaction (error: %+v)", err))
	}
	defer tx.Rollback()

	// Get the Pennsieve User
	user, err := p.q.GetUserById(ctx, int64(proposal.UserId))
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetUserById", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetUserById id: %d (error: %+v)", int64(proposal.UserId), err))
	}
	log.WithFields(log.Fields{"user": fmt.Sprintf("%+v", user)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	organization, err := p.q.GetOrganization(ctx, p.orgId)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetOrganization", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetOrganization id: %d (error: %+v)", p.orgId, err))
	}
	log.WithFields(log.Fields{"organization": fmt.Sprintf("%+v", organization)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// Add the Pennsieve User to the Workspace as a Guest
	orgUser, err := p.q.AddOrganizationUser(ctx, p.orgId, user.Id, model.Guest)
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddOrganizationUser", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddOrganizationUser orgId: %d userId: %d permBit: %d (error: %+v)", p.orgId, user.Id, model.Guest, err))
	}
	log.WithFields(log.Fields{"orgUser": fmt.Sprintf("%+v", orgUser)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	datasetStatus, err := p.q.GetDefaultDatasetStatus(ctx, int(p.orgId))
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDefaultDatasetStatus", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDefaultDatasetStatus organizationId: %d (error: %+v)", int(p.orgId), err))
	}
	log.WithFields(log.Fields{"datasetStatus": fmt.Sprintf("%+v", datasetStatus)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	dataUseAgreement, err := p.q.GetDefaultDataUseAgreement(ctx, int(p.orgId))
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDefaultDataUseAgreement", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDefaultDataUseAgreement organizationId: %d (error: %+v)", int(p.orgId), err))
	}
	log.WithFields(log.Fields{"dataUseAgreement": fmt.Sprintf("%+v", dataUseAgreement)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// create the dataset
	dataset, err := p.q.CreateDataset(ctx, pgdb.CreateDatasetParams{
		Name:                         proposal.Name,
		Description:                  "",
		Status:                       datasetStatus,
		AutomaticallyProcessPackages: false,
		License:                      "",
		Tags:                         nil,
		DataUseAgreement:             dataUseAgreement,
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "CreateDataset", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to CreateDataset (error: %+v)", err))
	}
	log.WithFields(log.Fields{"dataset": fmt.Sprintf("%+v", dataset)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	contributor, err := p.q.AddContributor(ctx, pgdb.NewContributor{
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		EmailAddress: user.Email,
		UserId:       user.Id,
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddContributor", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddContributor (error: %+v)", err))
	}
	log.WithFields(log.Fields{"contributor": fmt.Sprintf("%+v", contributor)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	datasetContributor, err := p.q.AddDatasetContributor(ctx, dataset, contributor)
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddDatasetContributor", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddDatasetContributor (error: %+v)", err))
	}
	log.WithFields(log.Fields{"datasetContributor": fmt.Sprintf("%+v", datasetContributor)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	datasetUser, err := p.q.AddDatasetUser(ctx, dataset, user, model.Owner)
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddDatasetUser", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddDatasetUser (error: %+v)", err))
	}
	log.WithFields(log.Fields{"datasetUser": fmt.Sprintf("%+v", datasetUser)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	if err = tx.Commit(); err != nil {
		log.WithFields(log.Fields{"failure": "Commit()", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to commit database transaction (error: %+v)", err))
	}

	return &CreatedDataset{
		User:         user,
		Organization: organization,
		Dataset:      dataset,
	}, nil
}
