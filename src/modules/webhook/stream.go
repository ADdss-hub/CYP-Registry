package webhook

import (
	"sync"
	"time"
)

// RegistryEventType 镜像仓库事件类型
type RegistryEventType string

const (
	// RegistryEventPush 镜像推送完成
	RegistryEventPush RegistryEventType = "push"
	// RegistryEventDelete 镜像删除完成
	RegistryEventDelete RegistryEventType = "delete"
)

// RegistryEvent 用于前端实时订阅的简化事件载荷
type RegistryEvent struct {
	Type       RegistryEventType `json:"type"`
	Repository string            `json:"repository"`
	Tag        string            `json:"tag,omitempty"`
	Digest     string            `json:"digest,omitempty"`
	ProjectID  string            `json:"projectId,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

var (
	registryEventMu       sync.Mutex
	registryEventChannels = make(map[chan RegistryEvent]struct{})
)

// SubscribeRegistryEvents 订阅 Registry 事件。
// 返回事件 channel 和一个用于取消订阅的函数。
func SubscribeRegistryEvents() (<-chan RegistryEvent, func()) {
	ch := make(chan RegistryEvent, 16)

	registryEventMu.Lock()
	registryEventChannels[ch] = struct{}{}
	registryEventMu.Unlock()

	cancel := func() {
		registryEventMu.Lock()
		if _, ok := registryEventChannels[ch]; ok {
			delete(registryEventChannels, ch)
			close(ch)
		}
		registryEventMu.Unlock()
	}

	return ch, cancel
}

// PublishRegistryEvent 将 RegistryEvent 广播给所有订阅者（最佳努力，非阻塞）。
func PublishRegistryEvent(event RegistryEvent) {
	registryEventMu.Lock()
	defer registryEventMu.Unlock()

	for ch := range registryEventChannels {
		select {
		case ch <- event:
		default:
			// 如果某个订阅者处理过慢，丢弃本次事件，避免阻塞整个系统
		}
	}
}
