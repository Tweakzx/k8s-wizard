package handlers

import (
	"net/http"
	"k8s-wizard/api/models"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:  "ok",
		Service: "k8s-wizard-api",
	})
}
