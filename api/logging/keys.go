package logging

// Structured log attribute keys. Use these constants rather than free-text
// field names so that one DataDog/CloudWatch query finds every line about a
// given request, user or proposal — the codebase previously spelled the same
// field "orgId", "orgNodeId", "OrgId" and "organizationNodeId" in different
// files, which made cross-file queries impossible.
//
// Canonical spelling is camelCase. One key per concept: an organization's
// numeric id is always KeyOrgID and its node id is always KeyOrgNodeID.
const (
	// KeyTraceID correlates every log line emitted while handling one request.
	// Taken from an inbound correlation header when present, otherwise
	// generated at the entrypoint. See handler.traceID.
	KeyTraceID = "traceId"

	// KeyRequestID is the API Gateway request id, kept alongside the trace id
	// so a log line can be tied back to an API Gateway access log entry.
	//
	// This is a per-hop, AWS-assigned id: API Gateway mints a fresh one for
	// every request it forwards, so it identifies this one hop and must not be
	// used to correlate a logical operation across services. Use KeyTraceID for
	// that.
	KeyRequestID = "requestId"

	// KeyAwsRequestID is the Lambda invocation id (lambdacontext.AwsRequestID).
	// It is a different identifier from KeyRequestID: that one belongs to the
	// API Gateway layer, this one to the underlying Lambda invocation, and the
	// two do not match. Logging both lets an operator cross-reference this
	// service's logs against AWS's own CloudWatch/X-Ray records for a specific
	// invocation. Like KeyRequestID it is per-hop, not a correlation id.
	KeyAwsRequestID = "awsRequestId"

	// Request routing.
	KeyMethod = "method"
	KeyRoute  = "route"
	KeyStatus = "status"

	// Identities.
	KeyOrgID       = "orgId"
	KeyOrgNodeID   = "orgNodeId"
	KeyUserID      = "userId"
	KeyNodeID      = "nodeId"
	KeyDatasetID   = "datasetId"
	KeyWorkspaceID = "workspaceId"
	KeyTeamID      = "teamId"

	// Domain objects.
	KeyProposal       = "proposal"
	KeyProposalStatus = "proposalStatus"
	KeyRepository     = "repository"
	KeyAction         = "action"

	// Datastores.
	KeyTable      = "table"
	KeyIndex      = "index"
	KeyCount      = "count"
	KeyS3Bucket   = "s3Bucket"
	KeyS3Key      = "s3Key"
	KeyStatement  = "statement"
	KeyItem       = "item"
	KeyQueryInput = "queryInput"

	// Notification / email.
	KeyRecipient  = "recipient"
	KeyRecipients = "recipients"
	KeySubject    = "subject"
	KeyMessageID  = "messageId"
	KeyTemplate   = "template"

	// KeyError carries the error value. Always use slog.Any(KeyError, err) so
	// the handler renders the error rather than a pre-formatted string.
	KeyError = "error"
)
