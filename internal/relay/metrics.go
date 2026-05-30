package relay

import (
	"sync/atomic"

	"github.com/coyoteXujie/mingsui/internal/protocol"
)

type metricsRecorder struct {
	activeConnections int64
	totalConnections  uint64
	uploadBytes       uint64
	downloadBytes     uint64
}

func (m *metricsRecorder) OpenConnection() {
	atomic.AddInt64(&m.activeConnections, 1)
	atomic.AddUint64(&m.totalConnections, 1)
}

func (m *metricsRecorder) CloseConnection() {
	atomic.AddInt64(&m.activeConnections, -1)
}

func (m *metricsRecorder) AddUploadBytes(n int64) {
	if n > 0 {
		atomic.AddUint64(&m.uploadBytes, uint64(n))
	}
}

func (m *metricsRecorder) AddDownloadBytes(n int64) {
	if n > 0 {
		atomic.AddUint64(&m.downloadBytes, uint64(n))
	}
}

func (m *metricsRecorder) Snapshot() protocol.Metrics {
	return protocol.Metrics{
		ActiveConnections: atomic.LoadInt64(&m.activeConnections),
		TotalConnections:  atomic.LoadUint64(&m.totalConnections),
		UploadBytes:       atomic.LoadUint64(&m.uploadBytes),
		DownloadBytes:     atomic.LoadUint64(&m.downloadBytes),
	}
}
