package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPlaygroundRequestedGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name        string
		bodyGroup   string
		headerGroup string
		want        string
	}{
		{name: "creative center header", bodyGroup: "old", headerGroup: "vip", want: "vip"},
		{name: "legacy playground body", bodyGroup: "legacy", want: "legacy"},
		{name: "default group", want: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/pg/images/async-generations", strings.NewReader(`{"model":"gpt-image-2","group":"`+tt.bodyGroup+`"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.Header.Set("X-Creative-Center-Group", tt.headerGroup)
			got, err := getPlaygroundRequestedGroup(c)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("group = %q, want %q", got, tt.want)
			}
		})
	}
}
