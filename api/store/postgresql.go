package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	pgdbModels "github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	pgdbQueries "github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/pennsieve/publishing-service/api/logging"
	"github.com/pennsieve/publishing-service/api/models"
)

const SystemTeamTypePublishers = "publishers"

type PennsievePublishingStore interface {
	GetProposalUser(ctx context.Context, userId int64) (*pgdbModels.User, error)
	GetRepositoryWorkspace(ctx context.Context, repository *models.Repository) (*pgdbModels.Organization, error)
	GetPublishingTeam(ctx context.Context, workspaceId int64) (*models.PublishingTeam, error)
	AddPublishingTeamToDataset(ctx context.Context, publishingTeam *models.PublishingTeam, dataset *pgdbModels.Dataset) error
	GetPublishingTeamMembers(ctx context.Context, repository *models.Repository) ([]models.Publisher, error)
	CreateDatasetForAcceptedProposal(ctx context.Context, proposal *models.DatasetProposal) (*CreatedDataset, error)
	GetWelcomeWorkspace(ctx context.Context) (*pgdbModels.Organization, error)
}

// NewPennsieveStore builds a store backed by db. logger is the request-scoped
// logger built at the entrypoint, held on the struct so no method has to reach
// for slog.Default. It returns an error (rather than panicking) when the
// initial transaction cannot be started, so that a transient database problem
// fails the one request instead of the process.
func NewPennsieveStore(logger *slog.Logger, db *sql.DB, orgId int64) (*pennsieveStore, error) {
	dbTx, err := db.BeginTx(context.TODO(), nil)
	if err != nil {
		logger.Error("db.BeginTx() failed constructing pennsieve store",
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return &pennsieveStore{
		logger: logger,
		orgId:  orgId,
		db:     db,
		q:      pgdbQueries.New(dbTx),
	}, nil
}

type pennsieveStore struct {
	logger *slog.Logger
	orgId  int64
	db     *sql.DB
	q      *pgdbQueries.Queries
}

type CreatedDataset struct {
	User         *pgdbModels.User
	Organization *pgdbModels.Organization
	Dataset      *pgdbModels.Dataset
}

func setOrgSearchPath(logger *slog.Logger, db *sql.DB, orgId int64) error {
	// Set Search Path to organization
	ctx := context.Background()
	_, err := db.ExecContext(ctx, fmt.Sprintf("SET search_path = \"%d\";", orgId))
	if err != nil {
		logger.Error("unable to set search_path to organization schema",
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Any(logging.KeyError, err))
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
		if err = setOrgSearchPath(p.logger, p.db, orgId); err != nil {
			return err
		}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		p.logger.Error("db.BeginTx() failed",
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Any(logging.KeyError, err))
		return err
	}

	q := pgdbQueries.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.Error("tx.Rollback() failed after a failed transaction",
				slog.Int64(logging.KeyOrgID, orgId),
				slog.Any(logging.KeyError, rbErr))
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		p.logger.Error("tx.Commit() failed",
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Any(logging.KeyError, err))
		return err
	}
	return nil
}

func (p *pennsieveStore) ExecPennsieveStoreTx(ctx context.Context, orgId int64, fn func(store *pennsieveStore) error) error {
	var err error

	// if organization id was provided, then set search path
	if orgId > 0 {
		if err = setOrgSearchPath(p.logger, p.db, orgId); err != nil {
			return err
		}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		p.logger.Error("db.BeginTx() failed",
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Any(logging.KeyError, err))
		return err
	}

	err = fn(p)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.Error("tx.Rollback() failed after a failed transaction",
				slog.Int64(logging.KeyOrgID, orgId),
				slog.Any(logging.KeyError, rbErr))
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		p.logger.Error("tx.Commit() failed",
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Any(logging.KeyError, err))
		return err
	}
	return nil
}

func (p *pennsieveStore) GetProposalUser(ctx context.Context, userId int64) (*pgdbModels.User, error) {
	return p.q.GetUserById(ctx, userId)
}

func (p *pennsieveStore) GetRepositoryWorkspace(ctx context.Context, repository *models.Repository) (*pgdbModels.Organization, error) {
	return p.q.GetOrganizationByNodeId(ctx, repository.OrganizationNodeId)
}

func (p *pennsieveStore) GetWelcomeWorkspace(ctx context.Context) (*pgdbModels.Organization, error) {
	return p.q.GetOrganizationBySlug(ctx, "welcome_to_pennsieve")
}

func (p *pennsieveStore) GetPublishingTeam(ctx context.Context, workspaceId int64) (*models.PublishingTeam, error) {

	queryStr := `select o.id as org_id,
		o.name as org_name,
		ot.team_id,
		t.name as team_name,
		ot.permission_bit,
		ot.system_team_type,
		t.node_id as team_node_id
		from pennsieve.organizations o
		join pennsieve.organization_team ot on ot.organization_id = o.id
		join pennsieve.teams t on t.id = ot.team_id
		where o.id = $1
		and ot.system_team_type = $2`

	var publishingTeam models.PublishingTeam
	row := p.db.QueryRowContext(ctx, queryStr, workspaceId, SystemTeamTypePublishers)
	err := row.Scan(
		&publishingTeam.WorkspaceId,
		&publishingTeam.WorkspaceName,
		&publishingTeam.TeamId,
		&publishingTeam.TeamName,
		&publishingTeam.PermissionBit,
		&publishingTeam.SystemTeamType,
		&publishingTeam.TeamNodeId,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			p.logger.Error("no publishing team found for workspace",
				slog.Int64(logging.KeyWorkspaceID, workspaceId))
		} else {
			p.logger.Error("failed to query publishing team for workspace",
				slog.Int64(logging.KeyWorkspaceID, workspaceId),
				slog.Any(logging.KeyError, err))
		}
		return nil, err
	}

	return &publishingTeam, nil
}

func (p *pennsieveStore) AddPublishingTeamToDataset(ctx context.Context, publishingTeam *models.PublishingTeam, dataset *pgdbModels.Dataset) error {
	statement := `INSERT INTO "%d".dataset_team
					(dataset_id, team_id, permission_bit, role)
					VALUES ($1, $2, $3, $4);`

	statement2 := fmt.Sprintf(statement, publishingTeam.WorkspaceId)

	_, err := p.db.ExecContext(
		ctx,
		statement2,
		dataset.Id,
		publishingTeam.TeamId,
		pgdbModels.Administer,
		strings.ToLower(role.Manager.String()),
	)

	return err
}

func (p *pennsieveStore) GetPublishingTeamMembers(ctx context.Context, repository *models.Repository) ([]models.Publisher, error) {
	queryStr := "select " +
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

	rows, err := p.db.QueryContext(ctx, queryStr, repository.OrganizationNodeId)
	if err != nil {
		p.logger.Error("failed to query publishing team members",
			slog.String(logging.KeyOrgNodeID, repository.OrganizationNodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}
	defer rows.Close()

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
			p.logger.Error("rows.Scan() failed reading a publishing team member",
				slog.String(logging.KeyOrgNodeID, repository.OrganizationNodeId),
				slog.Any(logging.KeyError, err))
		} else {
			publishers = append(publishers, publisher)
		}
	}
	if err := rows.Err(); err != nil {
		p.logger.Error("failed iterating publishing team members",
			slog.String(logging.KeyOrgNodeID, repository.OrganizationNodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return publishers, nil
}

func (p *pennsieveStore) CreateDatasetForAcceptedProposal(ctx context.Context, proposal *models.DatasetProposal) (*CreatedDataset, error) {
	var err error

	// Get the Pennsieve User
	user, err := p.q.GetUserById(ctx, int64(proposal.UserId))
	if err != nil {
		p.logger.Error("GetUserById() failed",
			slog.Int64(logging.KeyUserID, int64(proposal.UserId)),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetUserById: %w", err)
	}

	// Get the Organization
	organization, err := p.q.GetOrganization(ctx, p.orgId)
	if err != nil {
		p.logger.Error("GetOrganization() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetOrganization: %w", err)
	}

	// Add the Pennsieve User to the Workspace as a Guest
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddOrganizationUser(ctx, p.orgId, user.Id, pgdbModels.Guest)
		return err
	})
	if err != nil {
		p.logger.Error("AddOrganizationUser() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Int64(logging.KeyUserID, user.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to AddOrganizationUser: %w", err)
	}
	orgUser, err := p.q.GetOrganizationUser(ctx, p.orgId, user.Id)
	if err != nil {
		p.logger.Error("GetOrganizationUser() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Int64(logging.KeyUserID, user.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetOrganizationUser: %w", err)
	}
	p.logger.Debug("added user to workspace",
		slog.Int64(logging.KeyOrgID, p.orgId),
		slog.Int64(logging.KeyUserID, orgUser.UserId))

	// get the default dataset status
	datasetStatus, err := p.q.GetDefaultDatasetStatus(ctx, int(p.orgId))
	if err != nil {
		p.logger.Error("GetDefaultDatasetStatus() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetDefaultDatasetStatus: %w", err)
	}

	// get the default data use agreement
	dataUseAgreement, err := p.q.GetDefaultDataUseAgreement(ctx, int(p.orgId))
	if err != nil {
		p.logger.Error("GetDefaultDataUseAgreement() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetDefaultDataUseAgreement: %w", err)
	}

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
		p.logger.Error("CreateDataset() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to CreateDataset: %w", err)
	}
	ds, err := p.q.GetDatasetByName(ctx, proposal.Name)
	if err != nil {
		p.logger.Error("GetDatasetByName() failed",
			slog.Int64(logging.KeyOrgID, p.orgId),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetDatasetByName: %w", err)
	}

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
		p.logger.Error("AddContributor() failed",
			slog.Int64(logging.KeyUserID, user.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to AddContributor: %w", err)
	}
	contributor, err := p.q.GetContributorByUserId(ctx, user.Id)
	if err != nil {
		p.logger.Error("GetContributorByUserId() failed",
			slog.Int64(logging.KeyUserID, user.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetContributorByUserId: %w", err)
	}

	// attach the contributor to the dataset
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddDatasetContributor(ctx, ds, contributor)
		return err
	})
	if err != nil {
		p.logger.Error("AddDatasetContributor() failed",
			slog.Int64(logging.KeyDatasetID, ds.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to AddDatasetContributor: %w", err)
	}
	datasetContributor, err := p.q.GetDatasetContributor(ctx, ds.Id, contributor.Id)
	if err != nil {
		p.logger.Error("GetDatasetContributor() failed",
			slog.Int64(logging.KeyDatasetID, ds.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetDatasetContributor: %w", err)
	}
	p.logger.Debug("attached contributor to dataset",
		slog.Int64(logging.KeyDatasetID, ds.Id),
		slog.Int64("contributorId", datasetContributor.ContributorId))

	// add the user to the dataset as the owner
	err = p.ExecStoreTx(ctx, p.orgId, func(store *pgdbQueries.Queries) error {
		_, err := store.AddDatasetUser(ctx, ds, user, role.Owner)
		return err
	})
	if err != nil {
		p.logger.Error("AddDatasetUser() failed",
			slog.Int64(logging.KeyDatasetID, ds.Id),
			slog.Int64(logging.KeyUserID, user.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to AddDatasetUser: %w", err)
	}
	datasetUser, err := p.q.GetDatasetUser(ctx, ds, user)
	if err != nil {
		p.logger.Error("GetDatasetUser() failed",
			slog.Int64(logging.KeyDatasetID, ds.Id),
			slog.Int64(logging.KeyUserID, user.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetDatasetUser: %w", err)
	}
	p.logger.Debug("added user to dataset as owner",
		slog.Int64(logging.KeyDatasetID, ds.Id),
		slog.Int64(logging.KeyUserID, datasetUser.UserId))

	// add Publishers team to the newly created dataset
	publishingTeam, err := p.GetPublishingTeam(ctx, organization.Id)
	if err != nil {
		p.logger.Error("GetPublishingTeam() failed",
			slog.Int64(logging.KeyOrgID, organization.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to GetPublishingTeam: %w", err)
	}

	err = p.ExecPennsieveStoreTx(ctx, organization.Id, func(store *pennsieveStore) error {
		return store.AddPublishingTeamToDataset(ctx, publishingTeam, ds)
	})
	if err != nil {
		p.logger.Error("AddPublishingTeamToDataset() failed",
			slog.Int64(logging.KeyOrgID, organization.Id),
			slog.Int64(logging.KeyDatasetID, ds.Id),
			slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to AddPublishingTeamToDataset: %w", err)
	}

	return &CreatedDataset{
		User:         user,
		Organization: organization,
		Dataset:      ds,
	}, nil
}
