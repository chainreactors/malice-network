package errs

import (
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotFoundTarget = errors.New("target not found")
	ErrDisableOutput  = errors.New("output disabled")
)

// grpc
var (
	// ErrInvalidSessionID - Invalid Session ID in request
	ErrInvalidSessionID = status.Error(codes.InvalidArgument, "Invalid session ID")

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
	ErrNotFoundPipeline    = status.Error(codes.NotFound, "Pipeline not found")
	ErrNotFoundClientName  = status.Error(codes.NotFound, "Client name not found")
	ErrNotFoundTaskContent = status.Error(codes.NotFound, "Task content not found")
	ErrTaskIndexExceed     = status.Error(codes.NotFound, "task index id exceed total")
	//ErrInvalidBeaconTaskCancelState = status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid task state, must be '%s' to cancel", models.PENDING))
)
