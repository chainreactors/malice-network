package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *Server) ListSessions(ctx context.Context, req *clientpb.ListSessionsRequest) (*clientpb.ListSessionsResponse, error) {
	options, err := sessionListOptionsFromRequest(req)
	if err != nil {
		return nil, err
	}

	result, err := db.ListSessionsPageContext(ctx, options)
	if err != nil {
		return nil, sessionListRPCError("list sessions", err)
	}
	protobufSessions := result.Sessions.ToProtobuf()
	return &clientpb.ListSessionsResponse{
		Sessions:      protobufSessions.GetSessions(),
		FilteredTotal: result.Total,
		Page:          int32(result.Page),
		PageSize:      int32(result.PageSize),
		Stats:         sessionStatsToProtobuf(result.Stats),
	}, nil
}

func (rpc *Server) GetSessionStats(ctx context.Context, req *clientpb.SessionStatsRequest) (*clientpb.SessionStats, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	stats, err := db.GetSessionStatsContext(ctx, db.SessionStatsOptions{
		Group:  req.GetGroup(),
		Search: req.GetSearch(),
	})
	if err != nil {
		return nil, sessionListRPCError("get session stats", err)
	}
	return sessionStatsToProtobuf(*stats), nil
}

func sessionStatsToProtobuf(stats db.SessionStats) *clientpb.SessionStats {
	return &clientpb.SessionStats{
		Total:   stats.Total,
		Alive:   stats.Alive,
		Offline: stats.Offline,
	}
}

func (rpc *Server) ListSessionGroups(ctx context.Context, req *clientpb.Empty) (*clientpb.SessionGroups, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	groups, err := db.ListSessionGroupsContext(ctx)
	if err != nil {
		return nil, sessionListRPCError("list session groups", err)
	}
	return &clientpb.SessionGroups{Groups: groups}, nil
}

func (rpc *Server) GetSessionTrend(ctx context.Context, req *clientpb.Empty) (*clientpb.SessionTrend, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	points, err := db.GetSessionTrendContext(ctx, time.Now())
	if err != nil {
		return nil, sessionListRPCError("get session trend", err)
	}
	protobufPoints := make([]*clientpb.SessionTrendPoint, 0, len(points))
	for _, point := range points {
		protobufPoints = append(protobufPoints, &clientpb.SessionTrendPoint{
			BucketStartUnix: point.BucketStartUnix,
			Count:           point.Count,
		})
	}
	return &clientpb.SessionTrend{Points: protobufPoints}, nil
}

func sessionListOptionsFromRequest(req *clientpb.ListSessionsRequest) (db.SessionListOptions, error) {
	if req == nil {
		return db.SessionListOptions{}, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.GetPage() < 0 {
		return db.SessionListOptions{}, status.Error(codes.InvalidArgument, "page must be at least 1 or zero for the default")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > int32(db.MaxSessionListPageSize) {
		return db.SessionListOptions{}, status.Errorf(
			codes.InvalidArgument,
			"page_size must be between 1 and %d or zero for the default",
			db.MaxSessionListPageSize,
		)
	}

	listStatus, err := sessionListStatusFromProto(req.GetStatus())
	if err != nil {
		return db.SessionListOptions{}, err
	}
	options := db.SessionListOptions{
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
		Status:   listStatus,
		Group:    req.GetGroup(),
		Search:   req.GetSearch(),
	}

	if req.GetSortField() == clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_OPERATIONAL {
		if !isKnownSessionListSortDirection(req.GetSortDirection()) {
			return db.SessionListOptions{}, status.Errorf(
				codes.InvalidArgument,
				"unsupported session sort direction %d",
				req.GetSortDirection(),
			)
		}
		return options, nil
	}

	sortField, err := sessionListSortFieldFromProto(req.GetSortField())
	if err != nil {
		return db.SessionListOptions{}, err
	}
	sortDirection, err := sessionListSortDirectionFromProto(req.GetSortDirection())
	if err != nil {
		return db.SessionListOptions{}, err
	}
	options.Sort = &db.SessionListSort{
		Field:     sortField,
		Direction: sortDirection,
	}
	return options, nil
}

func sessionListStatusFromProto(value clientpb.SessionListStatus) (db.SessionListStatus, error) {
	switch value {
	case clientpb.SessionListStatus_SESSION_LIST_STATUS_ALL:
		return db.SessionListStatusAll, nil
	case clientpb.SessionListStatus_SESSION_LIST_STATUS_ALIVE:
		return db.SessionListStatusAlive, nil
	case clientpb.SessionListStatus_SESSION_LIST_STATUS_OFFLINE:
		return db.SessionListStatusOffline, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported session status %d", value)
	}
}

func sessionListSortFieldFromProto(value clientpb.SessionListSortField) (db.SessionListSortField, error) {
	switch value {
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_STATUS:
		return db.SessionListSortFieldIsAlive, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_SESSION_ID:
		return db.SessionListSortFieldSessionID, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_GROUP_NAME:
		return db.SessionListSortFieldGroup, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_NOTE:
		return db.SessionListSortFieldNote, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_LISTENER_ID:
		return db.SessionListSortFieldListenerID, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_PIPELINE_ID:
		return db.SessionListSortFieldPipelineID, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_TARGET:
		return db.SessionListSortFieldTarget, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_CREATED_AT:
		return db.SessionListSortFieldCreatedAt, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_LAST_CHECKIN:
		return db.SessionListSortFieldLastCheckin, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_PROFILE_NAME:
		return db.SessionListSortFieldProfileName, nil
	case clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_TYPE:
		return db.SessionListSortFieldType, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported session sort field %d", value)
	}
}

func sessionListSortDirectionFromProto(value clientpb.SessionListSortDirection) (db.SessionListSortDirection, error) {
	switch value {
	case clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_ASC:
		return db.SessionListSortAscending, nil
	case clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_DESC:
		return db.SessionListSortDescending, nil
	case clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_UNSPECIFIED:
		return "", status.Error(codes.InvalidArgument, "sort_direction must be ASC or DESC for an explicit sort field")
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported session sort direction %d", value)
	}
}

func isKnownSessionListSortDirection(value clientpb.SessionListSortDirection) bool {
	switch value {
	case clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_UNSPECIFIED,
		clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_ASC,
		clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_DESC:
		return true
	default:
		return false
	}
}

func sessionListRPCError(operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}
	if errors.Is(err, db.ErrInvalidSessionListOptions) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Errorf(codes.Internal, "%s: %v", operation, err)
}
