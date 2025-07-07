package errs

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpc
var (
	// ErrInvalidSessionID - Invalid Session ID in request
	ErrInvalidSessionID = status.Error(codes.InvalidArgument, "Invalid session ID")
	ErrInvalidateTarget = status.Error(codes.InvalidArgument, "target not validate")
	// ErrMissingRequestField - Returned when a request does not contain a  implantpb.Request
	ErrMissingRequestField = status.Error(codes.InvalidArgument, "Missing session request field")
	// ErrAsyncNotSupported - Unsupported mode / command type
	ErrAsyncNotSupported = status.Error(codes.Unavailable, "Async not supported for this command")
	// ErrDatabaseFailure - Generic database failure error (real error is logged)
	ErrDatabaseFailure = status.Error(codes.Internal, "Database operation failed")

	// ErrInvalidName - Invalid name
	ErrInvalidName     = status.Error(codes.InvalidArgument, "Invalid session name, alphanumerics and _-. only")
	ErrNotFoundSession = status.Error(codes.NotFound, "Session ID not found")
	ErrNotFoundTask    = status.Error(codes.NotFound, "Task ID not found")

	ErrNotFoundListener    = status.Error(codes.NotFound, "Listener not found")
	ErrNotFoundClientName  = status.Error(codes.NotFound, "Client name not found")
	ErrNotFoundTaskContent = status.Error(codes.NotFound, "Task content not found")
	ErrTaskIndexExceed     = status.Error(codes.NotFound, "task index id exceed total")

	ErrWorkflowFailed      = status.Error(codes.Unknown, "workflow failed")
	ErrorWorkflowNotActive = status.Error(codes.Unknown, "workflow not active")
	ErrorDockerNotActive   = status.Error(codes.Unknown, "docker not active")
	ErrNotFoundArtifact    = status.Error(codes.NotFound, "Artifact not found")
	//ErrInvalidBeaconTaskCancelState = status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid task state, must be '%s' to cancel", models.PENDING))

	ErrNotFoundGithubConfig = status.Error(codes.NotFound, "Github config not found")
	ErrNotFoundNotifyConfig = status.Error(codes.NotFound, "Notify config not found")

	ErrPlartFormNotSupport = status.Error(codes.Unimplemented, "Platform not support")
	ErrOBJCOPYFailed       = status.Error(codes.Unavailable, "OBJCOPY FAILED")
	ErrSrdiFailed          = status.Error(codes.Unavailable, "srdi FAILED")

	ErrSouceUnable          = status.Error(codes.Unavailable, "no build source available")
	ErrSaasUnable           = status.Error(codes.Unavailable, "saas server has a error")
	ErrLicenseTokenNotFound = status.Error(codes.NotFound, "License token not found")
)
