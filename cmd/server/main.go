package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	netserver "github.com/lulaide/gomc/pkg/network/server"
	"github.com/lulaide/gomc/pkg/tick"
)

func main() {
	addr := flag.String("addr", ":25565", "listen address")
	motd := flag.String("motd", "GoMC 1.6.4 rewrite", "server MOTD")
	maxPlayers := flag.Int("max-players", 20, "max players shown in server list")
	onlineMode := flag.Bool("online-mode", false, "enable online-mode login verification (not fully implemented yet)")
	viewDistance := flag.Int("view-distance", 10, "server view distance in chunks (3-15)")
	persistWorld := flag.Bool("persist-world", true, "enable Anvil world persistence")
	worldDir := flag.String("world-dir", "world", "world directory for region files")
	autosaveTicks := flag.Int64("autosave-ticks", 6000, "autosave interval in ticks (20 TPS)")
	flag.Parse()

	status := netserver.NewStatusServer(netserver.StatusConfig{
		ListenAddress: *addr,
		MOTD:          *motd,
		MaxPlayers:    *maxPlayers,
		VersionName:   "1.6.4",
		OnlineMode:    *onlineMode,
		ViewDistance:  *viewDistance,
		PersistWorld:  *persistWorld,
		WorldDir:      *worldDir,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- status.ListenAndServe()
	}()

	fmt.Printf("gomc server started: %s\n", status.String())

	var tickCount atomic.Int64
	loop := &tick.ServerLoop{
		Running: true,
		StartServer: func() bool {
			return true
		},
		Tick: func() {
			count := tickCount.Add(1)
			status.AdvanceWorldTime(1)
			status.TickChunkInhabitedTime()
			status.TickMobSpawning()
			status.TickProjectiles()
			status.TickDroppedItems()
			status.TickPlayerInfo()
			if *persistWorld && *autosaveTicks > 0 && count%*autosaveTicks == 0 {
				if err := status.SaveWorldDirty(); err != nil {
					fmt.Printf("autosave failed: %v\n", err)
				}
			}
			if count%20 == 0 {
				// Keep status player count synchronized with future player manager integration.
				status.SetCurrentPlayers(status.CurrentPlayers())
			}
		},
		Sleep: time.Sleep,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			fmt.Printf("gomc server shutting down on signal: %s\n", sig.String())
			loop.Running = false
		case err := <-errCh:
			if err != nil {
				fmt.Printf("network listener stopped: %v\n", err)
			}
			loop.Running = false
		}
	}()

	loop.Run()
	if err := status.Close(); err != nil {
		fmt.Printf("shutdown save failed: %v\n", err)
	}
	fmt.Println("gomc server stopped")
}
