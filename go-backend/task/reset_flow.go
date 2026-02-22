package task

import (
	"flux-panel/go-backend/model"
	"flux-panel/go-backend/pkg"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

func StartResetFlowTask(db *gorm.DB) {
	go func() {
		for {
			now := time.Now()
			// Schedule next run at 00:00:05
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
			time.Sleep(time.Until(next))

			log.Println("[ResetFlowTask] Starting daily flow reset...")
			resetFlow(db)
		}
	}()
}

func resetFlow(db *gorm.DB) {
	today := time.Now()
	dayOfMonth := today.Day()
	dayOfWeek := int(today.Weekday()) // 0=Sunday, 6=Saturday
	daysInMonth := daysInCurrentMonth(today)

	// 1. Reset user flow by periodic schedule
	// Monthly reset: flow_reset_type=1 AND (flow_reset_day=today OR (flow_reset_day>daysInMonth AND today is last day))
	var usersMonthly []model.User
	db.Where("flow_reset_type = 1 AND status = 1 AND (flow_reset_day = ? OR (flow_reset_day > ? AND ? = ?))",
		dayOfMonth, daysInMonth, dayOfMonth, daysInMonth).
		Find(&usersMonthly)

	// Weekly reset: flow_reset_type=2 AND flow_reset_day=weekday
	var usersWeekly []model.User
	db.Where("flow_reset_type = 2 AND status = 1 AND flow_reset_day = ?", dayOfWeek).
		Find(&usersWeekly)

	usersToReset := append(usersMonthly, usersWeekly...)
	for _, user := range usersToReset {
		db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
			"in_flow":       0,
			"out_flow":      0,
			"xray_in_flow":  0,
			"xray_out_flow": 0,
		})
		log.Printf("[ResetFlowTask] Reset user flow: userId=%d, user=%s, type=%d, day=%d", user.ID, user.User, user.FlowResetType, user.FlowResetDay)
	}

	// 2. Reset user_tunnel flow by periodic schedule
	var tunnelsMonthly []model.UserTunnel
	db.Where("flow_reset_type = 1 AND status = 1 AND (flow_reset_day = ? OR (flow_reset_day > ? AND ? = ?))",
		dayOfMonth, daysInMonth, dayOfMonth, daysInMonth).
		Find(&tunnelsMonthly)

	var tunnelsWeekly []model.UserTunnel
	db.Where("flow_reset_type = 2 AND status = 1 AND flow_reset_day = ?", dayOfWeek).
		Find(&tunnelsWeekly)

	tunnelsToReset := append(tunnelsMonthly, tunnelsWeekly...)
	for _, ut := range tunnelsToReset {
		db.Model(&model.UserTunnel{}).Where("id = ?", ut.ID).Updates(map[string]interface{}{
			"in_flow":  0,
			"out_flow": 0,
		})
		log.Printf("[ResetFlowTask] Reset user_tunnel flow: id=%d, userId=%d, tunnelId=%d", ut.ID, ut.UserId, ut.TunnelId)
	}

	// 3. Check expired users - pause their forwards and disable account
	nowMs := time.Now().UnixMilli()
	var expiredUsers []model.User
	db.Where("exp_time > 0 AND exp_time <= ? AND status = 1", nowMs).Find(&expiredUsers)

	for _, user := range expiredUsers {
		pauseUserForwards(db, user.ID)
		db.Model(&model.User{}).Where("id = ?", user.ID).Update("status", 0)
		log.Printf("[ResetFlowTask] Disabled expired user: userId=%d, user=%s", user.ID, user.User)
	}

	// 4. Check expired user_tunnels - pause their forwards and disable permission
	var expiredUTs []model.UserTunnel
	db.Where("exp_time > 0 AND exp_time <= ? AND status = 1", nowMs).Find(&expiredUTs)

	for _, ut := range expiredUTs {
		pauseUserTunnelForwards(db, ut.UserId, ut.TunnelId)
		db.Model(&model.UserTunnel{}).Where("id = ?", ut.ID).Update("status", 0)
		log.Printf("[ResetFlowTask] Disabled expired user_tunnel: id=%d, userId=%d, tunnelId=%d", ut.ID, ut.UserId, ut.TunnelId)
	}

	log.Println("[ResetFlowTask] Daily flow reset completed")
}

func pauseUserForwards(db *gorm.DB, userId int64) {
	var forwards []model.Forward
	db.Where("user_id = ? AND status = 1", userId).Find(&forwards)

	for _, fwd := range forwards {
		pauseSingleForward(db, &fwd)
	}
}

func pauseUserTunnelForwards(db *gorm.DB, userId int64, tunnelId int64) {
	var forwards []model.Forward
	db.Where("user_id = ? AND tunnel_id = ? AND status = 1", userId, tunnelId).Find(&forwards)

	for _, fwd := range forwards {
		pauseSingleForward(db, &fwd)
	}
}

func pauseSingleForward(db *gorm.DB, fwd *model.Forward) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ResetFlowTask] Failed to pause forward %d: %v", fwd.ID, r)
		}
	}()

	var tunnel model.Tunnel
	if err := db.First(&tunnel, fwd.TunnelId).Error; err != nil {
		return
	}

	var inNode model.Node
	if err := db.First(&inNode, tunnel.InNodeId).Error; err != nil {
		return
	}

	var ut model.UserTunnel
	db.Where("user_id = ? AND tunnel_id = ?", fwd.UserId, tunnel.ID).First(&ut)

	serviceName := fmt.Sprintf("%d_%d_%d", fwd.ID, fwd.UserId, ut.ID)

	if fwd.ListenIp != "" && strings.Contains(fwd.ListenIp, ",") {
		pkg.PauseServiceMultiIP(inNode.ID, serviceName, fwd.ListenIp)
	} else {
		pkg.PauseService(inNode.ID, serviceName)
	}
	if tunnel.Type == 2 {
		var outNode model.Node
		if err := db.First(&outNode, tunnel.OutNodeId).Error; err == nil {
			pkg.PauseRemoteService(outNode.ID, serviceName)
		}
	}

	db.Model(&model.Forward{}).Where("id = ?", fwd.ID).Update("status", 0)
}

// daysInCurrentMonth returns the number of days in the current month.
func daysInCurrentMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}
