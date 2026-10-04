//go:build noswagger

package routes

import "github.com/gin-gonic/gin"

// initSwaggerRoute does nothing in a noswagger build; see swagger.go.
func initSwaggerRoute(_ *gin.Engine) {}
