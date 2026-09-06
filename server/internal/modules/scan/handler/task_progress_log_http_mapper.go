package handler

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

const taskProgressLogPageTokenPrefix = "afterId:"

func toTaskProgressLogQueryInput(query *dto.TaskProgressLogListQuery) (*scanapp.TaskProgressLogListQuery, error) {
	if query == nil {
		return nil, nil
	}
	afterID, err := decodeTaskProgressLogPageToken(query.PageToken)
	if err != nil {
		return nil, err
	}
	return &scanapp.TaskProgressLogListQuery{
		AfterID: afterID,
		Limit:   query.PageSize,
	}, nil
}

func toTaskProgressLogListOutput(scanID int, logs []scanapp.TaskProgressLogEntry, hasMore bool) dto.TaskProgressLogListResponse {
	results := make([]dto.TaskProgressLogResponse, 0, len(logs))
	for index := range logs {
		item := logs[index]
		results = append(results, dto.TaskProgressLogResponse{
			ID:        item.ID,
			Name:      httpdto.ScanTaskName(scanID, int(item.ID)),
			ScanID:    scanID,
			TaskID:    item.TaskID,
			Level:     item.Level,
			Content:   item.Content,
			CreatedAt: timeutil.ToUTC(item.CreatedAt),
		})
	}
	nextPageToken := ""
	// AIP list pagination signals the terminal page by omitting nextPageToken.
	if hasMore && len(logs) > 0 {
		nextPageToken = encodeTaskProgressLogPageToken(logs[len(logs)-1].ID)
	}
	return dto.TaskProgressLogListResponse{Results: results, NextPageToken: nextPageToken}
}

func encodeTaskProgressLogPageToken(afterID int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s%d", taskProgressLogPageTokenPrefix, afterID)))
}

func decodeTaskProgressLogPageToken(token string) (int64, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid pageToken")
	}
	value := string(decoded)
	if !strings.HasPrefix(value, taskProgressLogPageTokenPrefix) {
		return 0, fmt.Errorf("invalid pageToken")
	}
	afterID, err := strconv.ParseInt(strings.TrimPrefix(value, taskProgressLogPageTokenPrefix), 10, 64)
	if err != nil || afterID < 0 {
		return 0, fmt.Errorf("invalid pageToken")
	}
	return afterID, nil
}
