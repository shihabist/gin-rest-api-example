package account

import (
	accountDB "gin-rest-api-example/internal/account/database"
	"gin-rest-api-example/internal/account/model"
	"gin-rest-api-example/internal/config"
	"gin-rest-api-example/pkg/logging"
	"github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

var identityKey = "id"

type signIn struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func CurrentUser(c *gin.Context) (*model.Account, bool) {
	data, ok := c.Get(identityKey)
	if !ok {
		return nil, false
	}
	acc, ok := data.(*model.Account)
	return acc, ok
}

func NewAuthMiddleware(cfg *config.Config, accountDB accountDB.AccountDB) (*jwt.GinJWTMiddleware, error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "test zone",
		Key:         []byte(cfg.JwtConfig.Secret),
		Timeout:     time.Duration(cfg.JwtConfig.SessionTime) * time.Millisecond,
		MaxRefresh:  time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*model.Account); ok {
				return jwt.MapClaims{
					identityKey: v.Email,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			email := claims[identityKey].(string)
			logging.FromContext(c).Info("middleware.jwt.IdentityHandler", "email", email)
			acc, _ := accountDB.FindByEmail(c.Request.Context(), email)
			return acc
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var req signIn
			if err := c.ShouldBind(&req); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			logging.FromContext(c).Info("middleware.jwt.Authenticator", "email", req.Username)

			acc, err := accountDB.FindByEmail(c.Request.Context(), req.Username)
			// TODO : password hash matches
			if err != nil || acc.Password != req.Password || acc.Disabled {
				return nil, jwt.ErrFailedAuthentication
			}
			return &model.Account{
				ID:       acc.ID,
				Username: acc.Username,
				Email:    acc.Email,
				Bio:      acc.Bio,
				Image:    acc.Image,
			}, nil
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			logging.FromContext(c).Info("middleware.jwt.Authorizator", "data", data)
			if _, ok := data.(*model.Account); ok {
				return true
			}
			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			logging.FromContext(c).Info("middleware.jwt.Unauthorized", "code", code, "message", message)
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, gin.H{
				"code":   code,
				"token":  token,
				"expire": expire,
				"meta":   "meta",
			})
		},
		TokenLookup:   "header: Authorization",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})
}
