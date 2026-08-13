package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GroupPriceMonitorHandler struct {
	service *service.GroupPriceMonitorService
}

func NewGroupPriceMonitorHandler(svc *service.GroupPriceMonitorService) *GroupPriceMonitorHandler {
	return &GroupPriceMonitorHandler{service: svc}
}

func (h *GroupPriceMonitorHandler) Get(c *gin.Context) {
	cfg, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *GroupPriceMonitorHandler) Update(c *gin.Context) {
	var cfg service.GroupPriceMonitorConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "invalid group price monitor config: "+err.Error())
		return
	}
	updated, err := h.service.SetConfig(c.Request.Context(), &cfg)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}
