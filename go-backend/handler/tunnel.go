package handler

import (
	"flux-panel/go-backend/dto"
	"flux-panel/go-backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TunnelCreate(c *gin.Context) {
	var d dto.TunnelDto
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}

	if useNodePassMode() {
		ingress, ok := npFindMachineIDByNumeric(d.InNodeId)
		if !ok {
			c.JSON(http.StatusOK, dto.Err("找不到入口节点（nodepass machine）"))
			return
		}
		egress := ingress
		if d.OutNodeId != nil {
			if v, ok := npFindMachineIDByNumeric(*d.OutNodeId); ok {
				egress = v
			}
		}
		start := 18081
		if d.Flow > 0 {
			start = d.Flow
		}
		payload := map[string]interface{}{
			"name":             d.Name,
			"protocol":         "tcp",
			"ingressMachineId": ingress,
			"egressMachineId":  egress,
			"enabled":          true,
			"mappings": []map[string]interface{}{
				{"inPort": start, "outAddr": "127.0.0.1", "outPort": start},
			},
		}
		var out map[string]interface{}
		if err := npReq("POST", "/tunnels", payload, &out); err != nil {
			c.JSON(http.StatusOK, dto.Err("创建 nodepass 隧道失败: "+err.Error()))
			return
		}
		c.JSON(http.StatusOK, dto.Ok(out))
		return
	}

	c.JSON(http.StatusOK, service.CreateTunnel(d))
}

func TunnelList(c *gin.Context) {
	if useNodePassMode() {
		tunnels, err := npGetTunnels()
		if err != nil {
			c.JSON(http.StatusOK, dto.Err("获取 nodepass 隧道失败: "+err.Error()))
			return
		}
		rows := make([]map[string]interface{}, 0, len(tunnels))
		for _, t := range tunnels {
			row := map[string]interface{}{
				"id":            stableID(t.ID),
				"name":          t.Name,
				"inNodeId":      stableID(t.IngressMachineId),
				"outNodeId":     stableID(t.EgressMachineId),
				"protocol":      t.Protocol,
				"type":          1,
				"status":        map[bool]int{true: 1, false: 0}[t.Enabled],
				"interfaceName": "",
			}
			if len(t.Mappings) > 0 {
				row["portSta"] = t.Mappings[0].InPort
				row["portEnd"] = t.Mappings[len(t.Mappings)-1].InPort
			}
			rows = append(rows, row)
		}
		c.JSON(http.StatusOK, dto.Ok(rows))
		return
	}
	c.JSON(http.StatusOK, service.GetAllTunnels())
}

func TunnelUpdate(c *gin.Context) {
	var d dto.TunnelUpdateDto
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}

	if useNodePassMode() {
		tid, ok := npFindTunnelIDByNumeric(d.ID)
		if !ok {
			c.JSON(http.StatusOK, dto.Err("找不到 nodepass 隧道"))
			return
		}
		payload := map[string]interface{}{
			"tunnelId": tid,
			"name":     d.Name,
			"protocol": "tcp",
		}
		var out map[string]interface{}
		if err := npReq("PUT", "/tunnels", payload, &out); err != nil {
			c.JSON(http.StatusOK, dto.Err("更新 nodepass 隧道失败: "+err.Error()))
			return
		}
		c.JSON(http.StatusOK, dto.Ok(out))
		return
	}

	c.JSON(http.StatusOK, service.UpdateTunnel(d))
}

func TunnelDelete(c *gin.Context) {
	var d struct {
		ID int64 `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}

	if useNodePassMode() {
		tid, ok := npFindTunnelIDByNumeric(d.ID)
		if !ok {
			c.JSON(http.StatusOK, dto.Err("找不到 nodepass 隧道"))
			return
		}
		if err := npReq("DELETE", "/tunnels?tunnelId="+tid, nil, nil); err != nil {
			c.JSON(http.StatusOK, dto.Err("删除 nodepass 隧道失败: "+err.Error()))
			return
		}
		c.JSON(http.StatusOK, dto.OkMsg())
		return
	}

	c.JSON(http.StatusOK, service.DeleteTunnel(d.ID))
}

func TunnelUserAssign(c *gin.Context) {
	var d dto.UserTunnelDto
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}
	c.JSON(http.StatusOK, service.AssignUserTunnel(d))
}

func TunnelUserList(c *gin.Context) {
	var d struct {
		TunnelId *int64 `json:"tunnelId"`
		UserId   *int64 `json:"userId"`
	}
	c.ShouldBindJSON(&d)
	c.JSON(http.StatusOK, service.ListUserTunnels(d.TunnelId, d.UserId))
}

func TunnelUserRemove(c *gin.Context) {
	var d struct {
		ID int64 `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}
	c.JSON(http.StatusOK, service.RemoveUserTunnel(d.ID))
}

func TunnelUserUpdate(c *gin.Context) {
	var d dto.UserTunnelUpdateDto
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}
	c.JSON(http.StatusOK, service.UpdateUserTunnel(d))
}

func TunnelUserTunnel(c *gin.Context) {
	userId := GetUserId(c)
	roleId := GetRoleId(c)
	c.JSON(http.StatusOK, service.GetUserAccessibleTunnels(userId, roleId))
}

func TunnelUpdateOrder(c *gin.Context) {
	var d struct {
		Items []dto.OrderItem `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}
	c.JSON(http.StatusOK, service.UpdateTunnelOrder(d.Items))
}

func TunnelDiagnose(c *gin.Context) {
	var d struct {
		ID int64 `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusOK, dto.Err("参数错误"))
		return
	}
	c.JSON(http.StatusOK, service.DiagnoseTunnel(d.ID))
}
