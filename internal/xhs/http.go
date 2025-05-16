// Package xhs 提供与小红书相关的 HTTP 服务。
package xhs

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册小红书相关路由。
// router: gin 路由引擎，signer: 签名服务实例。
func RegisterRoutes(router *gin.Engine, signer *Signer) {
	router.POST("/sign", func(c *gin.Context) {
		var req SignParams
		if err := c.ShouldBindJSON(&req); err != nil {
			slog.Warn("/sign 参数解析失败", "err", err, "client_ip", c.ClientIP())
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
			return
		}
		slog.Info("/sign 请求", "uri", req.URI, "client_ip", c.ClientIP())
		ctx := c.Request.Context()
		res, err := signer.Sign(ctx, req)
		if err != nil {
			slog.Error("/sign 签名失败", "err", err, "uri", req.URI, "client_ip", c.ClientIP())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "签名失败: " + err.Error()})
			return
		}
		slog.Info("/sign 成功", "uri", req.URI, "x-s", res.XS, "x-t", res.XT, "client_ip", c.ClientIP())
		c.JSON(http.StatusOK, res)
	})
}
