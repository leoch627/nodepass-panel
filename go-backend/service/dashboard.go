package service

import (
	"flux-panel/go-backend/dto"
	"flux-panel/go-backend/model"
	"flux-panel/go-backend/pkg"
	"sort"
	"time"
)

// GetAdminDashboardStats returns aggregated stats for admin dashboard.
func GetAdminDashboardStats() dto.R {
	// Nodes — use live WS status for accurate online count
	var allNodes []model.Node
	DB.Find(&allNodes)
	totalNodes := int64(len(allNodes))
	var onlineNodes int64
	for _, n := range allNodes {
		if pkg.WS != nil && pkg.WS.IsNodeOnline(n.ID) {
			onlineNodes++
		}
	}

	// Users (non-admin)
	var totalUsers int64
	DB.Model(&model.User{}).Where("role_id != 0").Count(&totalUsers)

	// Forwards
	var totalForwards, activeForwards int64
	DB.Model(&model.Forward{}).Count(&totalForwards)
	DB.Model(&model.Forward{}).Where("status = 1").Count(&activeForwards)

	// Traffic history + today's traffic (computed from same snapshot data)
	trafficData := getTrafficData(0)
	xrayData := getXrayTrafficData(0)

	// Total accumulated traffic from all users (real-time, not snapshot-based)
	type flowSum struct {
		Total int64
	}
	var totalGost, totalXray flowSum
	DB.Model(&model.User{}).Where("role_id != 0").
		Select("COALESCE(SUM(in_flow + out_flow), 0) as total").Scan(&totalGost)
	DB.Model(&model.User{}).Where("role_id != 0").
		Select("COALESCE(SUM(xray_in_flow + xray_out_flow), 0) as total").Scan(&totalXray)

	// Top 5 users by traffic (GOST + Xray combined)
	type TopUser struct {
		Name     string `json:"name" gorm:"column:user"`
		Flow     int64  `json:"flow"`
		GostFlow int64  `json:"gostFlow"`
		XrayFlow int64  `json:"vFlow"`
	}
	var topUsers []TopUser
	DB.Model(&model.User{}).
		Select("`user`, (in_flow + out_flow + xray_in_flow + xray_out_flow) as flow, (in_flow + out_flow) as gost_flow, (xray_in_flow + xray_out_flow) as xray_flow").
		Where("role_id != 0").
		Order("flow DESC").
		Limit(5).
		Find(&topUsers)

	// Node list — reuse allNodes with live WS status
	nodeList := make([]map[string]interface{}, 0, len(allNodes))
	for _, n := range allNodes {
		status := n.Status
		if pkg.WS != nil && pkg.WS.IsNodeOnline(n.ID) {
			status = 1
		}
		nodeList = append(nodeList, map[string]interface{}{
			"id":       n.ID,
			"name":     n.Name,
			"serverIp": n.ServerIp,
			"status":   status,
			"version":  n.Version,
		})
	}

	return dto.Ok(map[string]interface{}{
		"nodes":              map[string]int64{"total": totalNodes, "online": onlineNodes},
		"users":              map[string]int64{"total": totalUsers},
		"forwards":           map[string]int64{"total": totalForwards, "active": activeForwards},
		"todayTraffic":       trafficData.todayFlow,
		"trafficHistory":     trafficData.history,
		"todayXrayTraffic":   xrayData.todayFlow,
		"xrayTrafficHistory": xrayData.history,
		"totalGostFlow":      totalGost.Total,
		"totalXrayFlow":      totalXray.Total,
		"topUsers":           topUsers,
		"nodeList":           nodeList,
	})
}

// GetUserDashboardStats returns stats for a regular user.
func GetUserDashboardStats(userId int64) dto.R {
	// User package info
	var user model.User
	if err := DB.First(&user, userId).Error; err != nil {
		return dto.Err("用户不存在")
	}

	packageInfo := map[string]interface{}{
		"flow":          user.Flow,
		"inFlow":        user.InFlow,
		"outFlow":       user.OutFlow,
		"xrayFlow":      user.XrayFlow,
		"xrayInFlow":    user.XrayInFlow,
		"xrayOutFlow":   user.XrayOutFlow,
		"num":           user.Num,
		"expTime":       user.ExpTime,
		"flowResetType": user.FlowResetType,
		"flowResetDay":  user.FlowResetDay,
	}

	// Forward count
	var forwardCount int64
	DB.Model(&model.Forward{}).Where("user_id = ?", userId).Count(&forwardCount)

	// Traffic history for this user (per-user snapshots for accurate data)
	gostData, xrayData := getUserFlowData(userId)

	return dto.Ok(map[string]interface{}{
		"package":            packageInfo,
		"forwards":           forwardCount,
		"trafficHistory":     gostData.history,
		"xrayTrafficHistory": xrayData.history,
	})
}

// trafficData holds both 24h traffic history and today's total.
type trafficData struct {
	history  []map[string]interface{}
	todayFlow int64
}

// getTrafficData returns 24h traffic history and today's total from statistics_forward_flow.
// Uses cumulative snapshots with delta computation (same approach as monitor page).
// If userId=0, aggregates all forwards; otherwise filters by user's forwards.
func getTrafficData(userId int64) trafficData {
	cutoff := time.Now().Unix() - 25*3600 // fetch one extra hour for delta computation

	var records []model.StatisticsForwardFlow
	query := DB.Where("record_time >= ?", cutoff).Order("record_time ASC")
	if userId > 0 {
		var forwardIds []int64
		DB.Model(&model.Forward{}).Where("user_id = ?", userId).Pluck("id", &forwardIds)
		if len(forwardIds) == 0 {
			return trafficData{history: buildEmptyTrafficHistory()}
		}
		query = query.Where("forward_id IN ?", forwardIds)
	}
	query.Find(&records)

	if len(records) == 0 {
		return trafficData{history: buildEmptyTrafficHistory()}
	}

	// Group by (forwardId, bucket) → last snapshot
	bucketSize := int64(3600)
	type fwBucketKey struct {
		ForwardId int64
		Bucket    int64
	}
	fwBucketSnapshot := make(map[fwBucketKey]int64) // total flow
	for _, r := range records {
		key := fwBucketKey{r.ForwardId, (r.RecordTime / bucketSize) * bucketSize}
		fwBucketSnapshot[key] = r.InFlow + r.OutFlow
	}

	// Collect unique forward IDs and sorted buckets
	fwIds := make(map[int64]bool)
	allBuckets := make(map[int64]bool)
	for k := range fwBucketSnapshot {
		fwIds[k.ForwardId] = true
		allBuckets[k.Bucket] = true
	}

	sortedBuckets := make([]int64, 0, len(allBuckets))
	for b := range allBuckets {
		sortedBuckets = append(sortedBuckets, b)
	}
	sort.Slice(sortedBuckets, func(i, j int) bool { return sortedBuckets[i] < sortedBuckets[j] })

	// Compute deltas per bucket
	actualCutoff := time.Now().Unix() - 24*3600
	bucketFlow := make(map[int64]int64) // bucket → total delta flow
	for fwId := range fwIds {
		var prev int64
		firstSeen := false
		for _, bt := range sortedBuckets {
			snap, ok := fwBucketSnapshot[fwBucketKey{fwId, bt}]
			if !ok {
				continue
			}
			if !firstSeen {
				prev = snap
				firstSeen = true
				continue
			}
			if bt < actualCutoff {
				prev = snap
				continue
			}
			delta := snap - prev
			if delta < 0 {
				delta = 0
			}
			bucketFlow[bt] += delta
			prev = snap
		}
	}

	// Build 24-hour result — return Unix timestamps, let frontend format in browser timezone
	now := time.Now()
	nowTs := now.Unix()
	result := make([]map[string]interface{}, 0, 24)
	for i := 23; i >= 0; i-- {
		bt := ((nowTs - int64(i)*3600) / bucketSize) * bucketSize
		result = append(result, map[string]interface{}{
			"time": bt,
			"flow": bucketFlow[bt],
		})
	}

	// Sum today's traffic — use UTC-based "last 24h" to avoid server-timezone dependency
	var todayTotal int64
	cutoff24h := nowTs - 24*3600
	for bt, flow := range bucketFlow {
		if bt >= cutoff24h {
			todayTotal += flow
		}
	}

	return trafficData{history: result, todayFlow: todayTotal}
}

// getXrayTrafficData returns 24h Xray traffic history and today's total from statistics_xray_flow.
// Uses cumulative snapshots with delta computation (same approach as GOST).
// If userId=0, aggregates all inbounds; otherwise filters by user's inbound IDs.
func getXrayTrafficData(userId int64) trafficData {
	cutoff := time.Now().Unix() - 25*3600

	var records []model.StatisticsXrayFlow
	query := DB.Where("record_time >= ?", cutoff).Order("record_time ASC")
	if userId > 0 {
		var inboundIds []int64
		DB.Model(&model.XrayClient{}).Where("user_id = ?", userId).Distinct("inbound_id").Pluck("inbound_id", &inboundIds)
		if len(inboundIds) == 0 {
			return trafficData{history: buildEmptyTrafficHistory()}
		}
		query = query.Where("inbound_id IN ?", inboundIds)
	}
	query.Find(&records)

	if len(records) == 0 {
		return trafficData{history: buildEmptyTrafficHistory()}
	}

	bucketSize := int64(3600)
	type ibBucketKey struct {
		InboundId int64
		Bucket    int64
	}
	ibBucketSnapshot := make(map[ibBucketKey]int64) // total flow (up+down)
	for _, r := range records {
		key := ibBucketKey{r.InboundId, (r.RecordTime / bucketSize) * bucketSize}
		ibBucketSnapshot[key] = r.UpFlow + r.DownFlow
	}

	ibIds := make(map[int64]bool)
	allBuckets := make(map[int64]bool)
	for k := range ibBucketSnapshot {
		ibIds[k.InboundId] = true
		allBuckets[k.Bucket] = true
	}

	sortedBuckets := make([]int64, 0, len(allBuckets))
	for b := range allBuckets {
		sortedBuckets = append(sortedBuckets, b)
	}
	sort.Slice(sortedBuckets, func(i, j int) bool { return sortedBuckets[i] < sortedBuckets[j] })

	actualCutoff := time.Now().Unix() - 24*3600
	bucketFlow := make(map[int64]int64)
	for ibId := range ibIds {
		var prev int64
		firstSeen := false
		for _, bt := range sortedBuckets {
			snap, ok := ibBucketSnapshot[ibBucketKey{ibId, bt}]
			if !ok {
				continue
			}
			if !firstSeen {
				prev = snap
				firstSeen = true
				continue
			}
			if bt < actualCutoff {
				prev = snap
				continue
			}
			delta := snap - prev
			if delta < 0 {
				delta = 0
			}
			bucketFlow[bt] += delta
			prev = snap
		}
	}

	now := time.Now()
	nowTs := now.Unix()
	result := make([]map[string]interface{}, 0, 24)
	for i := 23; i >= 0; i-- {
		bt := ((nowTs - int64(i)*3600) / bucketSize) * bucketSize
		result = append(result, map[string]interface{}{
			"time": bt,
			"flow": bucketFlow[bt],
		})
	}

	var todayTotal int64
	cutoff24h := nowTs - 24*3600
	for bt, flow := range bucketFlow {
		if bt >= cutoff24h {
			todayTotal += flow
		}
	}

	return trafficData{history: result, todayFlow: todayTotal}
}

// getUserFlowData returns 24h GOST and Xray traffic history for a specific user
// using per-user flow snapshots (accurate, not inflated by shared inbounds).
func getUserFlowData(userId int64) (gostData trafficData, xrayData trafficData) {
	cutoff := time.Now().Unix() - 25*3600

	var records []model.StatisticsUserFlow
	DB.Where("user_id = ? AND record_time >= ?", userId, cutoff).
		Order("record_time ASC").
		Find(&records)

	if len(records) == 0 {
		return trafficData{history: buildEmptyTrafficHistory()}, trafficData{history: buildEmptyTrafficHistory()}
	}

	bucketSize := int64(3600)
	type bucketSnap struct {
		GostFlow int64
		XrayFlow int64
	}
	bucketSnapshot := make(map[int64]bucketSnap)
	for _, r := range records {
		bt := (r.RecordTime / bucketSize) * bucketSize
		bucketSnapshot[bt] = bucketSnap{r.GostFlow, r.XrayFlow}
	}

	sortedBuckets := make([]int64, 0, len(bucketSnapshot))
	for b := range bucketSnapshot {
		sortedBuckets = append(sortedBuckets, b)
	}
	sort.Slice(sortedBuckets, func(i, j int) bool { return sortedBuckets[i] < sortedBuckets[j] })

	actualCutoff := time.Now().Unix() - 24*3600
	gostBucketFlow := make(map[int64]int64)
	xrayBucketFlow := make(map[int64]int64)

	var prevGost, prevXray int64
	firstSeen := false
	for _, bt := range sortedBuckets {
		snap := bucketSnapshot[bt]
		if !firstSeen {
			prevGost = snap.GostFlow
			prevXray = snap.XrayFlow
			firstSeen = true
			continue
		}
		if bt < actualCutoff {
			prevGost = snap.GostFlow
			prevXray = snap.XrayFlow
			continue
		}
		gostDelta := snap.GostFlow - prevGost
		xrayDelta := snap.XrayFlow - prevXray
		if gostDelta < 0 {
			gostDelta = 0
		}
		if xrayDelta < 0 {
			xrayDelta = 0
		}
		gostBucketFlow[bt] = gostDelta
		xrayBucketFlow[bt] = xrayDelta
		prevGost = snap.GostFlow
		prevXray = snap.XrayFlow
	}

	now := time.Now()
	nowTs := now.Unix()
	gostHistory := make([]map[string]interface{}, 0, 24)
	xrayHistory := make([]map[string]interface{}, 0, 24)
	for i := 23; i >= 0; i-- {
		bt := ((nowTs - int64(i)*3600) / bucketSize) * bucketSize
		gostHistory = append(gostHistory, map[string]interface{}{
			"time": bt,
			"flow": gostBucketFlow[bt],
		})
		xrayHistory = append(xrayHistory, map[string]interface{}{
			"time": bt,
			"flow": xrayBucketFlow[bt],
		})
	}

	var gostTotal, xrayTotal int64
	cutoff24h := nowTs - 24*3600
	for bt, flow := range gostBucketFlow {
		if bt >= cutoff24h {
			gostTotal += flow
		}
	}
	for bt, flow := range xrayBucketFlow {
		if bt >= cutoff24h {
			xrayTotal += flow
		}
	}

	return trafficData{history: gostHistory, todayFlow: gostTotal},
		trafficData{history: xrayHistory, todayFlow: xrayTotal}
}

func buildEmptyTrafficHistory() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 24)
	nowTs := time.Now().Unix()
	bucketSize := int64(3600)
	for i := 23; i >= 0; i-- {
		bt := ((nowTs - int64(i)*3600) / bucketSize) * bucketSize
		result = append(result, map[string]interface{}{
			"time": bt,
			"flow": int64(0),
		})
	}
	return result
}
