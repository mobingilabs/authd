package v1

import (
	"net/http"

	"github.com/labstack/echo"
)

type apiv1 struct {
	e *echo.Echo
	g *echo.Group
}

func (a *apiv1) token(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func NewApiV1(e *echo.Echo) *apiv1 {
	g := e.Group("/api/v1")
	api := &apiv1{
		e: e,
		g: g,
	}

	g.POST("/token", api.token)

	return api
}
