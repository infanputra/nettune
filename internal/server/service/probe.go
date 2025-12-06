package service

import (
	"github.com/jtsang4/nettune/internal/server/adapter"
	"github.com/jtsang4/nettune/internal/shared/types"
	"go.uber.org/zap"
)

// ProbeService handles probe-related operations on the server side
type ProbeService struct {
	adapter *adapter.SystemAdapter
	logger  *zap.Logger
}

// NewProbeService creates a new ProbeService
func NewProbeService(adapter *adapter.SystemAdapter, logger *zap.Logger) *ProbeService {
	return &ProbeService{
		adapter: adapter,
		logger:  logger,
	}
}

// GetServerInfo returns server information
func (s *ProbeService) GetServerInfo() (*types.ServerInfo, error) {
	return s.adapter.SysInfo.GetServerInfo()
}
