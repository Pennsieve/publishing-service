package models

type Publisher struct {
	WorkspaceId                int64
	WorkspaceName              string
	TeamName                   string
	TeamId                     int64
	TeamPermissionBit          int64
	UserId                     int64
	UserName                   string
	UserEmailAddress           string
	UserTeamPermissionBit      int64
	UserWorkspacePermissionBit int64
}
