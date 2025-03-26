package scheduler

import (
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
)

type SchedulerImpl struct {
}

func NewScheduler() Scheduler {
	return &SchedulerImpl{}
}

func (s *SchedulerImpl) ScheduleInference(req *http.Request, pods []*corev1.Pod) (*corev1.Pod, error) {
	if len(pods) == 0 {
		return nil, fmt.Errorf("pods shouldn't be empty")
	}

	// For POC purpose, use first pod directly
	return pods[0], nil
}

func (s *SchedulerImpl) RunFilterPlugins(pods []*corev1.Pod) ([]*corev1.Pod, error) {
	return nil, nil
}

func (s *SchedulerImpl) RunScorePlugins(pods []*corev1.Pod) (map[string]int, error) {
	return nil, nil
}
