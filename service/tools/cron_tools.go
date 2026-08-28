package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/er1cw00/claw.go/service/cron"
)

// CronTool exposes cron job management to the agent.
type CronTool struct {
	BaseTool
}

// NewCronTool creates a new cron tool.
func NewCronTool() *CronTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["add", "list", "remove"],
				"description": "Action to perform"
			},
			"message": {
				"type": "string",
				"description": "Reminder message (for add)"
			},
			"every_seconds": {
				"type": "integer",
				"description": "Interval in seconds (for recurring tasks)"
			},
			"cron_expr": {
				"type": "string",
				"description": "Cron expression like '0 9 * * *' (for scheduled tasks)"
			},
			"at": {
				"type": "string",
				"description": "ISO datetime for one-time execution (e.g. '2026-02-12T10:30:00')"
			},
			"job_id": {
				"type": "string",
				"description": "Job ID (for remove)"
			}
		},
		"required": ["action"]
	}`)
	return &CronTool{
		BaseTool: BaseTool{
			name:        "cron",
			description: "manage scheduled cron jobs: add, list, remove",
			parameters:  schema,
		},
	}
}

func (t *CronTool) Name() string {
	return t.name
}

func (t *CronTool) Description() string {
	return t.description
}

func (t *CronTool) ParametersSchema() json.RawMessage {
	return t.parameters
}

func (t *CronTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
	}
}

func (t *CronTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)
	switch action {
	case "add":
		return t.addJob(args)
	case "list":
		return t.listJobs()
	case "remove":
		return t.removeJob(args)
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

func (t *CronTool) addJob(args map[string]interface{}) (string, error) {
	message, _ := args["message"].(string)
	if message == "" {
		return "", errors.New("message is required for add")
	}

	schedule, err := t.parseSchedule(args)
	if err != nil {
		return "", err
	}

	name := message
	if len(name) > 40 {
		name = name[:40]
	}

	deleteAfterRun := schedule.Kind == cron.ScheduleKindAt
	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)
	if channel == "" {
		channel = "cron"
	}
	if chatID == "" {
		chatID = "default"
	}

	job, err := cron.GetService().AddJob(name, schedule, message, false, channel, chatID, deleteAfterRun)
	if err != nil {
		return "", fmt.Errorf("failed to add job: %w", err)
	}

	data, _ := json.MarshalIndent(job, "", "  ")
	return fmt.Sprintf("Added cron job:\n%s", string(data)), nil
}

func (t *CronTool) listJobs() (string, error) {
	jobs := cron.GetService().ListJobs(false)
	if len(jobs) == 0 {
		return "No cron jobs found.", nil
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal jobs: %w", err)
	}
	return string(data), nil
}

func (t *CronTool) removeJob(args map[string]interface{}) (string, error) {
	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return "", errors.New("job_id is required for remove")
	}
	if ok := cron.GetService().RemoveJob(jobID); !ok {
		return fmt.Sprintf("Job %s not found.", jobID), nil
	}
	return fmt.Sprintf("Job %s removed.", jobID), nil
}

func (t *CronTool) parseSchedule(args map[string]interface{}) (cron.CronSchedule, error) {
	atRaw, hasAt := args["at"].(string)
	everySeconds, hasEvery := getInt(args, "every_seconds")
	cronExpr, hasCron := args["cron_expr"].(string)

	if hasAt && atRaw != "" {
		atMs, err := parseAtTime(atRaw)
		if err != nil {
			return cron.CronSchedule{}, err
		}
		return cron.CronSchedule{
			Kind: cron.ScheduleKindAt,
			AtMs: atMs,
		}, nil
	}
	if hasEvery && everySeconds > 0 {
		return cron.CronSchedule{
			Kind:    cron.ScheduleKindEvery,
			EveryMs: int64(everySeconds) * 1000,
		}, nil
	}
	if hasCron && cronExpr != "" {
		return cron.CronSchedule{
			Kind: cron.ScheduleKindCron,
			Expr: cronExpr,
		}, nil
	}
	return cron.CronSchedule{}, errors.New("one of at, every_seconds or cron_expr is required for add")
}

func parseAtTime(raw string) (int64, error) {
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return ms, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("cannot parse 'at' time: %s", raw)
}

func getInt(args map[string]interface{}, key string) (int, bool) {
	switch v := args[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}
