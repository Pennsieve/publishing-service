package models

type PublishingTeam struct {
	WorkspaceId    int64
	WorkspaceName  string
	TeamId         int64
	TeamName       string
	PermissionBit  int64
	SystemTeamType string
	TeamNodeId     string
}

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
