package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dataset/role"
	pgdbModels "github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	pgdbQueries "github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/pennsieve/publishing-service/api/models"
	log "github.com/sirupsen/logrus"
)

type PennsievePublishingStore interface {
	GetProposalUser(ctx context.Context, userId int64) (*pgdbModels.User, error)
	GetRepositoryWorkspace(ctx context.Context, repository *models.Repository) (*pgdbModels.Organization, error)
	GetPublishingTeam(ctx context.Context, repository *models.Repository) ([]models.Publisher, error)
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
		q:     pgdbQueries.New(dbTx),
	}
}

type pennsieveStore struct {
	orgId int64
	db    *sql.DB
	q     *pgdbQueries.Queries
}

type CreatedDataset struct {
	User         *pgdbModels.User
	Organization *pgdbModels.Organization
	Dataset      *pgdbModels.Dataset
}

func setOrgSearchPath(db *sql.DB, orgId int64) error {
	// Set Search Path to organization
	ctx := context.Background()
	_, err := db.ExecContext(ctx, fmt.Sprintf("SET search_path = \"%d\";", orgId))
	if err != nil {
		log.Error(fmt.Sprintf("Unable to set search_path to %d.", orgId))
		return err
	}

	return err
}

// ExecStoreTx will execute the function fn, passing in a new SQLStore instance that
// is backed by a database transaction. Any methods fn runs against the passed in SQLStore will run
// in this transaction. If fn returns a non-nil error, the transaction will be rolled back.
// Otherwise, the transaction will be committed.
func (p *pennsieveStore) ExecStoreTx(ctx context.Context, orgId int64, fn func(store *pgdbQueries.Queries) error) error {
	var err error

	// if organization id was provided, then set search path
	if orgId > 0 {
		if err = setOrgSearchPath(p.db, orgId); err != nil {
			return err
		}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := pgdbQueries.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

func (p *pennsieveStore) GetProposalUser(ctx context.Context, userId int64) (*pgdbModels.User, error) {
	return p.q.GetUserById(ctx, userId)
}

func (p *pennsieveStore) GetRepositoryWorkspace(ctx context.Context, repository *models.Repository) (*pgdbModels.Organization, error) {
	return p.q.GetOrganizationByNodeId(ctx, repository.OrganizationNodeId)
}

func (p *pennsieveStore) GetPublishingTeam(ctx context.Context, repository *models.Repository) ([]models.Publisher, error) {
	query := "select " +
		"  o.id as Workspace_Id, " +
		"  o.name as Workspace_Name, " +
		"  t.name as team_name, " +
		"  ot.team_id as team_id, " +
		"  ot.permission_bit as team_permission_bit, " +
		"  tu.user_id as user_id, " +
		"  u.first_name || ' ' || u.last_name as user_name, " +
		"  u.email as user_email_address, " +
		"  tu.permission_bit as user_team_permission_bit, " +
		"  ou.permission_bit as user_workspace_permission_bit " +
		"from pennsieve.organizations o " +
		"join pennsieve.organization_team ot on o.id=ot.organization_id " +
		"join pennsieve.teams t on ot.team_id=t.id " +
		"join pennsieve.team_user tu on t.id=tu.team_id " +
		"join pennsieve.users u on tu.user_id=u.id " +
		"join pennsieve.organization_user ou on u.id=ou.user_id and o.id=ou.organization_id " +
		"where o.node_id=$1 " +
		"and ot.system_team_type='publishers';"

	rows, err := p.db.QueryContext(ctx, query, repository.OrganizationNodeId)
	if err != nil {
		log.WithFields(log.Fields{"QueryContext": "failed", "error": fmt.Sprintf("%+v", err)}).Error("GetPublishingTeam()")
		return nil, err
	}

	var publishers []models.Publisher
	for rows.Next() {
		var publisher models.Publisher
		err := rows.Scan(
			&publisher.WorkspaceId,
			&publisher.WorkspaceName,
			&publisher.TeamName,
			&publisher.TeamId,
			&publisher.TeamPermissionBit,
			&publisher.UserId,
			&publisher.UserName,
			&publisher.UserEmailAddress,
			&publisher.UserTeamPermissionBit,
			&publisher.UserWorkspacePermissionBit,
		)
		if err != nil {
			log.WithFields(log.Fields{"status": "error", "error": fmt.Sprintf("%+v", err)}).Error("rows.Scan()")
		} else {
			publishers = append(publishers, publisher)
		}
	}

	return publishers, nil
}

func (p *pennsieveStore) CreateDatasetForAcceptedProposal(ctx context.Context, proposal *models.DatasetProposal) (*CreatedDataset, error) {
	var err error

	// Get the Pennsieve User
	user, err := p.q.GetUserById(ctx, int64(proposal.UserId))
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetUserById", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetUserById id: %d (error: %+v)", int64(proposal.UserId), err))
	}
	log.WithFields(log.Fields{"user": fmt.Sprintf("%+v", user)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// Get the Organization
	organization, err := p.q.GetOrganization(ctx, p.orgId)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetOrganization", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetOrganization id: %d (error: %+v)", p.orgId, err))
	}
	log.WithFields(log.Fields{"organization": fmt.Sprintf("%+v", organization)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// Add the Pennsieve User to the Workspace as a Guest
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddOrganizationUser(ctx, p.orgId, user.Id, pgdbModels.Guest)
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddOrganizationUser", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddOrganizationUser orgId: %d userId: %d permBit: %d (error: %+v)", p.orgId, user.Id, pgdbModels.Guest, err))
	}
	orgUser, err := p.q.GetOrganizationUser(ctx, p.orgId, user.Id)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetOrganizationUser", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetOrganizationUser orgId: %d userId: %d (error: %+v)", p.orgId, user.Id, err))
	}
	log.WithFields(log.Fields{"orgUser": fmt.Sprintf("%+v", orgUser)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// get the default dataset status
	datasetStatus, err := p.q.GetDefaultDatasetStatus(ctx, int(p.orgId))
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDefaultDatasetStatus", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDefaultDatasetStatus organizationId: %d (error: %+v)", int(p.orgId), err))
	}
	log.WithFields(log.Fields{"datasetStatus": fmt.Sprintf("%+v", datasetStatus)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// get the default data use agreement
	dataUseAgreement, err := p.q.GetDefaultDataUseAgreement(ctx, int(p.orgId))
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDefaultDataUseAgreement", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDefaultDataUseAgreement organizationId: %d (error: %+v)", int(p.orgId), err))
	}
	log.WithFields(log.Fields{"dataUseAgreement": fmt.Sprintf("%+v", dataUseAgreement)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// create the dataset
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.CreateDataset(ctx, pgdbQueries.CreateDatasetParams{
			Name:                         proposal.Name,
			Description:                  "",
			Status:                       datasetStatus,
			AutomaticallyProcessPackages: false,
			License:                      "",
			Tags:                         nil,
			DataUseAgreement:             dataUseAgreement,
		})
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "CreateDataset", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to CreateDataset (error: %+v)", err))
	}
	ds, err := p.q.GetDatasetByName(ctx, proposal.Name)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDatasetByName", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDatasetByName (error: %+v)", err))
	}
	log.WithFields(log.Fields{"ds": fmt.Sprintf("%+v", ds)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// create the contributor record
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddContributor(ctx, pgdbQueries.NewContributor{
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			EmailAddress: user.Email,
			UserId:       user.Id,
		})
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddContributor", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddContributor (error: %+v)", err))
	}
	contributor, err := p.q.GetContributorByUserId(ctx, user.Id)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetContributorByUserId", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetContributorByUserId (error: %+v)", err))
	}
	log.WithFields(log.Fields{"contributor": fmt.Sprintf("%+v", contributor)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// attach the contributor to the dataset
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddDatasetContributor(ctx, ds, contributor)
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddDatasetContributor", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddDatasetContributor (error: %+v)", err))
	}
	datasetContributor, err := p.q.GetDatasetContributor(ctx, ds.Id, contributor.Id)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDatasetContributor", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDatasetContributor (error: %+v)", err))
	}
	log.WithFields(log.Fields{"datasetContributor": fmt.Sprintf("%+v", datasetContributor)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	// add the user to the dataset as the owner
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddDatasetUser(ctx, ds, user, role.Owner)
		return err
	})
	if err != nil {
		log.WithFields(log.Fields{"failure": "AddDatasetUser", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to AddDatasetUser (error: %+v)", err))
	}
	datasetUser, err := p.q.GetDatasetUser(ctx, ds, user)
	if err != nil {
		log.WithFields(log.Fields{"failure": "GetDatasetUser", "err": fmt.Sprintf("%+v", err)}).Error("pennsieveStore.CreateDatasetForAcceptedProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to GetDatasetUser (error: %+v)", err))
	}
	log.WithFields(log.Fields{"datasetUser": fmt.Sprintf("%+v", datasetUser)}).Debug("pennsieveStore.CreateDatasetForAcceptedProposal()")

	return &CreatedDataset{
		User:         user,
		Organization: organization,
		Dataset:      ds,
	}, nil
}
