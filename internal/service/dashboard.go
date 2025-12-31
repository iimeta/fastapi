// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IDashboard interface {
		// Subscription
		Subscription(ctx context.Context) (*model.DashboardSubscriptionRes, error)
		// Usage
		Usage(ctx context.Context) (*model.DashboardUsageRes, error)
	}
)

var (
	localDashboard IDashboard
)

func Dashboard() IDashboard {
	if localDashboard == nil {
		panic("implement not found for interface IDashboard, forgot register?")
	}
	return localDashboard
}

func RegisterDashboard(i IDashboard) {
	localDashboard = i
}
