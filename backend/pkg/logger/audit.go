package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	AuditResultSuccess = "success"
	AuditResultFailed  = "failed"
)

var (
	auditMu     sync.Mutex
	auditOutput io.Writer = os.Stdout
)

func SetAuditOutput(w io.Writer) func() {
	auditMu.Lock()
	previous := auditOutput
	auditOutput = w
	auditMu.Unlock()

	return func() {
		auditMu.Lock()
		auditOutput = previous
		auditMu.Unlock()
	}
}

func Audit(c *gin.Context, event, result string, fields map[string]any) {
	entry := map[string]any{
		"type":       "audit",
		"event":      event,
		"result":     result,
		"request_id": c.GetString("request_id"),
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}

	if userID := c.GetString("user_id"); userID != "" {
		entry["user_id"] = userID
	}
	if role := c.GetString("user_role"); role != "" {
		entry["role"] = role
	}
	for key, value := range fields {
		if value == nil || value == "" {
			continue
		}
		entry[key] = value
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}

	auditMu.Lock()
	defer auditMu.Unlock()
	_, _ = fmt.Fprintln(auditOutput, string(payload))
}
