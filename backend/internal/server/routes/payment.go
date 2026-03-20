package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPaymentRoutes 注册支付相关路由
func RegisterPaymentRoutes(
	v1 *gin.RouterGroup,
	h *handler.PaymentHandler,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	// 易支付异步回调（无需认证）
	payment := v1.Group("/payment")
	{
		payment.POST("/notify", h.Notify)
	}

	// 需要认证的支付接口
	authenticated := v1.Group("/payment")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		authenticated.POST("/orders", h.CreateOrder)
		authenticated.GET("/orders", h.GetOrders)
		authenticated.POST("/orders/:id/cancel", h.CancelOrder)
		authenticated.POST("/orders/:id/pay", h.RetryPayment)
		authenticated.GET("/exchange-rate", h.GetExchangeRate)
		authenticated.GET("/limits", h.GetLimits)
	}
}
