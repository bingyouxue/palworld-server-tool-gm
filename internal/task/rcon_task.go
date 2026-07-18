package task

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/robfig/cron/v3"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
	"go.etcd.io/bbolt"
)

const rconTaskTagPrefix = "rcon-task:"

var rconExecutionMu sync.Mutex

// restartServerFunc is injected by the api package at startup.
// It must kill any surviving server processes and then relaunch the server.
// mode is "silent" or "cmd".
var restartServerFunc func(mode string) error
var restartFuncMu sync.RWMutex

// SetRestartFunc registers the callback that is invoked after a Shutdown RCON
// task fires, so the server is automatically relaunched.
func SetRestartFunc(fn func(mode string) error) {
	restartFuncMu.Lock()
	defer restartFuncMu.Unlock()
	restartServerFunc = fn
}

// isShutdownCommand returns true when the full RCON command begins with
// "Shutdown" (case-insensitive).
func isShutdownCommand(cmd string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(cmd)), "SHUTDOWN")
}

// shutdownSeconds parses the countdown from "Shutdown <N> …", defaulting to 60.
func shutdownSeconds(cmd string) int {
	parts := strings.Fields(cmd)
	if len(parts) >= 2 {
		var n int
		if _, err := fmt.Sscanf(parts[1], "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return 60
}

func normalizeStartMode(mode string) string {
	if mode == "cmd" {
		return "cmd"
	}
	return "silent"
}

// triggerAutoRestart waits for the server to stop, then calls the registered
// restart function. Runs in its own goroutine so it never blocks the task runner.
func triggerAutoRestart(shutdownCmd, mode string) {
	waitSecs := shutdownSeconds(shutdownCmd) + 12
	if waitSecs < 20 {
		waitSecs = 20
	}
	mode = normalizeStartMode(mode)
	logger.Infof("[AutoRestart] server shutting down, will restart in %d seconds using %s mode…\n", waitSecs, mode)
	time.Sleep(time.Duration(waitSecs) * time.Second)

	restartFuncMu.RLock()
	fn := restartServerFunc
	restartFuncMu.RUnlock()

	if fn == nil {
		logger.Warn("[AutoRestart] no restart function registered; server will NOT be restarted automatically\n")
		return
	}
	if err := fn(mode); err != nil {
		logger.Errorf("[AutoRestart] failed to restart server: %v\n", err)
		return
	}
	logger.Infof("[AutoRestart] server process launched successfully using %s mode\n", mode)
}

func ValidateCronExpression(expression string) error {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return errors.New("cron expression is required")
	}
	if _, err := cron.ParseStandard(expression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

func RegisterRconTask(db *bbolt.DB, rconTask database.RconTask) error {
	UnregisterRconTask(rconTask.UUID)
	if !rconTask.Enabled {
		return nil
	}
	if err := ValidateCronExpression(rconTask.Cron); err != nil {
		return err
	}
	scheduler := getScheduler()
	_, err := scheduler.NewJob(
		gocron.CronJob(rconTask.Cron, false),
		gocron.NewTask(ExecuteRconTask, db, rconTask.UUID),
		gocron.WithName(rconTask.Name),
		gocron.WithTags(rconTaskTagPrefix+rconTask.UUID),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	return err
}

func UnregisterRconTask(taskUUID string) {
	getScheduler().RemoveByTags(rconTaskTagPrefix + taskUUID)
}

func LoadRconTasks(db *bbolt.DB) error {
	tasks, err := service.ListRconTasks(db)
	if err != nil {
		return err
	}
	for _, rconTask := range tasks {
		if err := RegisterRconTask(db, rconTask); err != nil {
			logger.Errorf("Failed to schedule RCON task %s: %v\n", rconTask.UUID, err)
		}
	}
	return nil
}

func NextRconTaskRun(taskUUID string) *time.Time {
	tag := rconTaskTagPrefix + taskUUID
	for _, job := range getScheduler().Jobs() {
		for _, jobTag := range job.Tags() {
			if jobTag != tag {
				continue
			}
			next, err := job.NextRun()
			if err != nil || next.IsZero() {
				return nil
			}
			return &next
		}
	}
	return nil
}

func ExecuteRconTask(db *bbolt.DB, taskUUID string) error {
	return executeRconTask(db, taskUUID, tool.CustomCommand)
}

func executeRconTask(db *bbolt.DB, taskUUID string, execute func(string) (string, error)) error {
	rconExecutionMu.Lock()
	defer rconExecutionMu.Unlock()

	rconTask, err := service.GetRconTask(db, taskUUID)
	if err != nil {
		return err
	}
	rconCommand, err := service.GetRconCommand(db, rconTask.RconUUID)
	if err != nil {
		ranAt := time.Now()
		_ = service.UpdateRconTaskExecution(db, taskUUID, "failed", "", err.Error(), ranAt)
		return err
	}
	execCommand := strings.TrimSpace(strings.Join([]string{rconCommand.Command, rconTask.Content}, " "))
	result, executeErr := execute(execCommand)
	ranAt := time.Now()
	if executeErr != nil {
		if updateErr := service.UpdateRconTaskExecution(db, taskUUID, "failed", result, executeErr.Error(), ranAt); updateErr != nil {
			logger.Errorf("Failed to save RCON task result %s: %v\n", taskUUID, updateErr)
		}
		return executeErr
	}
	if err := service.UpdateRconTaskExecution(db, taskUUID, "success", result, "", ranAt); err != nil {
		return err
	}
	logger.Infof("Scheduled RCON task %s executed successfully\n", taskUUID)

	// If this task sent a Shutdown command, schedule an automatic restart in
	// a background goroutine after the server has had time to stop.
	if isShutdownCommand(execCommand) {
		go triggerAutoRestart(execCommand, rconTask.StartMode)
	}

	return nil
}
