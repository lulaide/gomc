package main

import (
	"fmt"
	"sync"

	netclient "github.com/lulaide/gomc/pkg/network/client"
)

type offlineGUIController struct {
	username string
	baseCfg  integratedOfflineConfig

	mu     sync.Mutex
	active *integratedOfflineServer
}

func newOfflineGUIController(username string, cfg integratedOfflineConfig) *offlineGUIController {
	return &offlineGUIController{
		username: username,
		baseCfg:  cfg,
	}
}

func (c *offlineGUIController) ConnectWorld(worldDir string) (*netclient.Session, error) {
	cfg := c.baseCfg
	cfg.WorldDir = worldDir

	nextServer, err := startIntegratedOfflineServer(cfg)
	if err != nil {
		return nil, err
	}
	nextSession, err := netclient.DialAndLogin(nextServer.Addr(), c.username)
	if err != nil {
		_ = nextServer.Close()
		return nil, err
	}

	c.mu.Lock()
	prev := c.active
	c.active = nextServer
	c.mu.Unlock()

	if prev != nil {
		_ = prev.Close()
	}

	fmt.Printf("offline mode: local world server started at %s (world=%s)\n", nextServer.Addr(), worldDir)
	fmt.Printf("connected to %s as %s\n", nextServer.Addr(), c.username)
	return nextSession, nil
}

func (c *offlineGUIController) Close() error {
	c.mu.Lock()
	active := c.active
	c.active = nil
	c.mu.Unlock()
	if active == nil {
		return nil
	}
	return active.Close()
}
