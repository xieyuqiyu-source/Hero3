// 本文件实现世界地图 HTTP 接口。
package api

import (
	"errors"
	"net/http"

	"hero3/internal/app/game"
)

// WorldMapView 返回世界地图玩家城池视野。
func (h *Handlers) WorldMapView(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetWorldMapView(
		playerID,
		queryInt(r, "centerX", -1),
		queryInt(r, "centerY", -1),
		queryInt(r, "radius", 0),
	)
	if err != nil {
		if errors.Is(err, game.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// WorldMapPlayerCityTarget 返回单个玩家城池目标详情。
func (h *Handlers) WorldMapPlayerCityTarget(w http.ResponseWriter, r *http.Request) {
	viewerID := r.URL.Query().Get("viewerId")
	if !h.requireOwnership(w, r, viewerID) {
		return
	}
	result, err := h.gameService.GetWorldMapPlayerCityTarget(viewerID, r.PathValue("playerId"))
	if err != nil {
		if errors.Is(err, game.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminWorldMapOccupancy 返回 GM 世界地图占用率统计。
func (h *Handlers) AdminWorldMapOccupancy(w http.ResponseWriter, r *http.Request) {
	result, err := h.gameService.AdminWorldMapOccupancy()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminWorldCoordinateCheck 返回 GM 查询的世界坐标占用状态。
func (h *Handlers) AdminWorldCoordinateCheck(w http.ResponseWriter, r *http.Request) {
	result, err := h.gameService.AdminCheckWorldCoordinate(
		queryInt(r, "x", -1),
		queryInt(r, "y", -1),
	)
	if err != nil {
		writeWorldMapAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminWorldPosition 返回 GM 查询的玩家世界坐标。
func (h *Handlers) AdminWorldPosition(w http.ResponseWriter, r *http.Request) {
	result, err := h.gameService.AdminGetWorldPosition(r.PathValue("playerId"))
	if err != nil {
		writeWorldMapAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// UpdateAdminWorldPosition 保存 GM 调整后的玩家世界坐标。
func (h *Handlers) UpdateAdminWorldPosition(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	result, err := h.gameService.AdminAssignWorldPosition(r.PathValue("playerId"), payload.X, payload.Y)
	if err != nil {
		writeWorldMapAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// writeWorldMapAdminError 转换 GM 世界地图错误为 HTTP 状态码。
func writeWorldMapAdminError(w http.ResponseWriter, err error) {
	if errors.Is(err, game.ErrPlayerNotFound) {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	if errors.Is(err, game.ErrInvalidWorldCoordinate) {
		writeError(w, http.StatusBadRequest, "invalid or occupied world coordinate")
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}
