package desktop

import (
	"strings"
	"sync"
)

type LogBuffer struct {
	mu      sync.Mutex
	max     int
	lines   []string
	partial string
}

func NewLogBuffer(maxLines int) *LogBuffer {
	if maxLines <= 0 {
		maxLines = 200
	}
	return &LogBuffer{max: maxLines}
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	text := b.partial + string(p)
	b.partial = ""
	for {
		index := strings.IndexByte(text, '\n')
		if index < 0 {
			b.partial = text
			break
		}
		b.appendLocked(strings.TrimRight(text[:index], "\r"))
		text = text[index+1:]
	}
	return len(p), nil
}

func (b *LogBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	lines := make([]string, len(b.lines))
	copy(lines, b.lines)
	if b.partial != "" {
		lines = append(lines, b.partial)
	}
	return lines
}

func (b *LogBuffer) appendLocked(line string) {
	b.lines = append(b.lines, line)
	if len(b.lines) <= b.max {
		return
	}
	copy(b.lines, b.lines[len(b.lines)-b.max:])
	b.lines = b.lines[:b.max]
}
