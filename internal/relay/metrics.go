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
	_ = m.ReserveConnection(0)
	m.CommitConnection()
}

func (m *metricsRecorder) ReserveConnection(maxConnections int) bool {
	if maxConnections <= 0 {
		atomic.AddInt64(&m.activeConnections, 1)
		return true
	}

	limit := int64(maxConnections)
	for {
		active := atomic.LoadInt64(&m.activeConnections)
		if active >= limit {
			return false
		}
		if atomic.CompareAndSwapInt64(&m.activeConnections, active, active+1) {
			return true
		}
	}
}

func (m *metricsRecorder) CommitConnection() {
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
