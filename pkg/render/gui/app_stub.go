//go:build !cgo

package gui

import (
	"fmt"

	netclient "github.com/lulaide/gomc/pkg/network/client"
)

type Config struct {
	Width            int
	Height           int
	RenderDistance   int
	MouseSensitivity float64
	MoveSpeed        float64
	FPSLimitMode     int
	StartInMainMenu  bool
	CurrentWorld     string
	PlayWorld        func(worldDir string) (*netclient.Session, error)
}

func Run(session *netclient.Session, cfg Config) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	return fmt.Errorf("GUI client requires CGO_ENABLED=1 and an OpenGL/GLFW runtime")
}
