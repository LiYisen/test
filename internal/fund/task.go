package fund

import (
	"fmt"
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

type BacktestTask struct {
	ID          string      `json:"id"`
	FundID      string      `json:"fund_id"`
	FundName    string      `json:"fund_name"`
	StartDate   string      `json:"start_date"`
	EndDate     string      `json:"end_date"`
	Status      TaskStatus  `json:"status"`
	Progress    int         `json:"progress"`
	TotalSteps  int         `json:"total_steps"`
	CurrentStep string      `json:"current_step"`
	Message     string      `json:"message"`
	ResultID    string      `json:"result_id,omitempty"`
	Error       string      `json:"error,omitempty"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     *time.Time  `json:"end_time,omitempty"`
}

type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*BacktestTask
}

var globalTaskManager = &TaskManager{
	tasks: make(map[string]*BacktestTask),
}

func GetTaskManager() *TaskManager {
	return globalTaskManager
}

func (tm *TaskManager) CreateTask(fundID, fundName, startDate, endDate string) *BacktestTask {
	taskID := fmt.Sprintf("task_%s_%d", fundID, time.Now().UnixNano())
	task := &BacktestTask{
		ID:        taskID,
		FundID:    fundID,
		FundName:  fundName,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    TaskStatusPending,
		Progress:  0,
		StartTime: time.Now(),
	}

	tm.mu.Lock()
	tm.tasks[taskID] = task
	tm.mu.Unlock()

	return task
}

func (tm *TaskManager) GetTask(taskID string) (*BacktestTask, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasks[taskID]
	if !ok {
		return nil, false
	}
	return task, true
}

func (tm *TaskManager) UpdateProgress(taskID string, progress int, currentStep string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if task, ok := tm.tasks[taskID]; ok {
		task.Progress = progress
		task.CurrentStep = currentStep
		task.Status = TaskStatusRunning
	}
}

func (tm *TaskManager) SetTotalSteps(taskID string, total int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if task, ok := tm.tasks[taskID]; ok {
		task.TotalSteps = total
	}
}

func (tm *TaskManager) CompleteTask(taskID, resultID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if task, ok := tm.tasks[taskID]; ok {
		task.Status = TaskStatusCompleted
		task.Progress = 100
		task.ResultID = resultID
		task.Message = "回测完成"
		now := time.Now()
		task.EndTime = &now
	}
}

func (tm *TaskManager) FailTask(taskID, errMsg string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if task, ok := tm.tasks[taskID]; ok {
		task.Status = TaskStatusFailed
		task.Error = errMsg
		task.Message = "回测失败"
		now := time.Now()
		task.EndTime = &now
	}
}

func (tm *TaskManager) ListTasks() []*BacktestTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tasks := make([]*BacktestTask, 0, len(tm.tasks))
	for _, t := range tm.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (tm *TaskManager) CleanupOldTasks(maxAge time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	now := time.Now()
	for id, task := range tm.tasks {
		if task.EndTime != nil && now.Sub(*task.EndTime) > maxAge {
			delete(tm.tasks, id)
		}
	}
}
