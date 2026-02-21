package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	netserver "github.com/lulaide/gomc/pkg/network/server"
)

type integratedOfflineConfig struct {
	WorldDir      string
	PersistWorld  bool
	ViewDistance  int
	AutosaveTicks int64
}

type integratedOfflineServer struct {
	addr   string
	status *netserver.StatusServer

	persistWorld  bool
	autosaveTicks int64

	stop chan struct{}
	done chan struct{}

	closeOnce sync.Once
	closeErr  error
}

func (s *integratedOfflineServer) Addr() string {
	return s.addr
}

func (s *integratedOfflineServer) Close() error {
	s.closeOnce.Do(func() {
		close(s.stop)
		<-s.done
		s.closeErr = s.status.Close()
	})
	return s.closeErr
}

func startIntegratedOfflineServer(cfg integratedOfflineConfig) (*integratedOfflineServer, error) {
	addr, err := reserveLocalListenAddr()
	if err != nil {
		return nil, err
	}

	status := netserver.NewStatusServer(netserver.StatusConfig{
		ListenAddress: addr,
		MOTD:          "GoMC Offline World",
		MaxPlayers:    8,
		VersionName:   "1.6.4",
		OnlineMode:    false,
		ViewDistance:  cfg.ViewDistance,
		PersistWorld:  cfg.PersistWorld,
		WorldDir:      cfg.WorldDir,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- status.ListenAndServe()
	}()

	if err := waitForServerReady(addr, errCh, 3*time.Second); err != nil {
		_ = status.Close()
		return nil, err
	}

	server := &integratedOfflineServer{
		addr:          addr,
		status:        status,
		persistWorld:  cfg.PersistWorld,
		autosaveTicks: cfg.AutosaveTicks,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go server.tickLoop()
	return server, nil
}

func (s *integratedOfflineServer) tickLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	defer close(s.done)

	var tickCount int64
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			tickCount++
			s.status.AdvanceWorldTime(1)
			s.status.TickChunkInhabitedTime()
			s.status.TickMobSpawning()
			s.status.TickProjectiles()
			s.status.TickPlayerInfo()

			if s.persistWorld && s.autosaveTicks > 0 && tickCount%s.autosaveTicks == 0 {
				if err := s.status.SaveWorldDirty(); err != nil {
					fmt.Printf("[offline] autosave failed: %v\n", err)
				}
			}
		}
	}
}

func reserveLocalListenAddr() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()
	if closeErr := ln.Close(); closeErr != nil {
		return "", closeErr
	}
	return addr, nil
}

func waitForServerReady(addr string, errCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
			return fmt.Errorf("integrated server stopped before ready")
		default:
		}

		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for integrated server at %s", addr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
