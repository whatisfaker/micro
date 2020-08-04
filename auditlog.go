package micro

import (
	"github.com/gin-gonic/gin"
	auditlog "github.com/whatisfaker/gin-contrib/audit"
	"github.com/whatisfaker/zaptrace/log"
)

type audit struct {
	log *log.Factory
}

func (c *audit) ginaudit(name string) gin.HandlerFunc {
	return auditlog.MWAuditlog(name)
}

func (c *MSManager) GinAudit(name string) gin.HandlerFunc {
	return c.audit.ginaudit(name)
}
