// Model server scheduling
package scheduler

import (
	"net/http"

	corev1 "k8s.io/api/core/v1"
)

type Scheduler interface {
	ScheduleInference(req *http.Request, pods []*corev1.Pod) (*corev1.Pod, error)
	RunFilterPlugins(pods []*corev1.Pod) ([]*corev1.Pod, error)
	RunScorePlugins(pods []*corev1.Pod) (map[string]int, error)
}

type FilterPlugin interface {
	Filter([]*corev1.Pod) ([]*corev1.Pod, error)
}

type ScorePlugin interface {
	Score(pods []*corev1.Pod) (map[string]int, error)
}
