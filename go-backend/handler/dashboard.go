package handler

import (
	"flux-panel/go-backend/dto"
	"flux-panel/go-backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func DashboardStats(c *gin.Context) {
	if useNodePassMode() {
		machines, _ := npGetMachines()
		tunnels, _ := npGetTunnels()
		online := 0
		for _, m := range machines {
			if m.Status == "online" {
				online++
			}
		}
		active := 0
		for _, t := range tunnels {
			if t.Enabled {
				active++
			}
		}
		c.JSON(http.StatusOK, dto.Ok(map[string]interface{}{
			"nodes":    map[string]int64{"total": int64(len(machines)), "online": int64(online)},
			"forwards": map[string]int64{"total": int64(len(tunnels)), "active": int64(active)},
			"users":    map[string]int64{"total": 0},
		}))
		return
	}

	roleId, _ := c.Get("roleId")
	if roleId == 0 {
		c.JSON(http.StatusOK, service.GetAdminDashboardStats())
		return
	}
	userId := c.GetInt64("userId")
	c.JSON(http.StatusOK, service.GetUserDashboardStats(userId))
}
