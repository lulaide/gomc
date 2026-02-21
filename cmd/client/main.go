package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	netclient "github.com/lulaide/gomc/pkg/network/client"
	"github.com/lulaide/gomc/pkg/network/crypt"
	"github.com/lulaide/gomc/pkg/network/protocol"
	rendergui "github.com/lulaide/gomc/pkg/render/gui"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:25565", "server address")
	play := flag.Bool("play", false, "run interactive play client")
	cli := flag.Bool("cli", false, "use CLI play loop (debug mode)")
	status := flag.Bool("status", false, "run legacy status ping")
	offline := flag.Bool("offline", false, "start integrated offline server (singleplayer)")
	offlineWorld := flag.String("offline-world", "world-offline", "world directory for integrated offline mode")
	offlinePersist := flag.Bool("offline-persist", true, "persist integrated offline world")
	offlineViewDistance := flag.Int("offline-view-distance", 10, "view distance for integrated offline mode")
	offlineAutosaveTicks := flag.Int64("offline-autosave-ticks", 6000, "autosave interval for integrated offline mode in ticks (20 TPS)")
	guiWidth := flag.Int("gui-width", 1280, "GUI window width")
	guiHeight := flag.Int("gui-height", 720, "GUI window height")
	guiRenderDistance := flag.Int("gui-render-distance", 10, "GUI client render distance in chunks")
	guiMouseSensitivity := flag.Float64("gui-mouse-sensitivity", 0.14, "GUI mouse sensitivity")
	guiMoveSpeed := flag.Float64("gui-move-speed", 4.3, "GUI movement speed")
	guiFPSMode := flag.Int("gui-fps-mode", 1, "GUI framerate mode: 0=max(200), 1=balanced(120), 2=powersaver(35)")
	guiSkipMenu := flag.Bool("gui-skip-menu", false, "skip main menu and enter world directly")
	loginProbe := flag.Bool("login-probe", false, "run offline login probe instead of status ping")
	username := flag.String("username", "Steve", "username for login probe/play")
	flag.Parse()

	// Double-click default: launch offline singleplayer GUI menu.
	if !*play && !*status && !*loginProbe {
		*play = true
		*offline = true
	}

	if *offline && !*play {
		*play = true
	}

	if *play {
		if *cli {
			targetAddr := *addr
			var integrated *integratedOfflineServer
			var err error
			if *offline {
				integrated, err = startIntegratedOfflineServer(integratedOfflineConfig{
					WorldDir:      *offlineWorld,
					PersistWorld:  *offlinePersist,
					ViewDistance:  *offlineViewDistance,
					AutosaveTicks: *offlineAutosaveTicks,
				})
				if err != nil {
					fmt.Printf("start integrated offline server failed: %v\n", err)
					return
				}
				defer func() {
					if closeErr := integrated.Close(); closeErr != nil {
						fmt.Printf("offline server shutdown failed: %v\n", closeErr)
					}
				}()
				targetAddr = integrated.Addr()
				fmt.Printf("offline mode: local world server started at %s (world=%s)\n", targetAddr, *offlineWorld)
			}
			runPlayCLI(targetAddr, *username)
			return
		}

		cfg := rendergui.Config{
			Width:            *guiWidth,
			Height:           *guiHeight,
			RenderDistance:   *guiRenderDistance,
			MouseSensitivity: *guiMouseSensitivity,
			MoveSpeed:        *guiMoveSpeed,
			FPSLimitMode:     *guiFPSMode,
			StartInMainMenu:  !*guiSkipMenu,
		}

		if *offline {
			controller := newOfflineGUIController(*username, integratedOfflineConfig{
				WorldDir:      *offlineWorld,
				PersistWorld:  *offlinePersist,
				ViewDistance:  *offlineViewDistance,
				AutosaveTicks: *offlineAutosaveTicks,
			})
			defer func() {
				if closeErr := controller.Close(); closeErr != nil {
					fmt.Printf("offline server shutdown failed: %v\n", closeErr)
				}
			}()
			session, err := controller.ConnectWorld(*offlineWorld)
			if err != nil {
				fmt.Printf("start integrated offline world failed: %v\n", err)
				return
			}
			cfg.PlayWorld = controller.ConnectWorld
			cfg.CurrentWorld = *offlineWorld
			runPlayGUI(session, cfg)
			return
		}

		session, err := netclient.DialAndLogin(*addr, *username)
		if err != nil {
			fmt.Printf("connect/login failed: %v\n", err)
			return
		}
		runPlayGUI(session, cfg)
		return
	}
	if *loginProbe {
		runLoginProbe(*addr, *username)
		return
	}
	if *status {
		runStatusPing(*addr)
		return
	}
	runStatusPing(*addr)
}

func runStatusPing(addr string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("invalid addr %q: %v\n", addr, err)
		return
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		fmt.Printf("invalid port %q: %v\n", port, err)
		return
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("connect failed: %v\n", err)
		return
	}
	defer conn.Close()

	ping := &protocol.Packet254ServerPing{
		ReadSuccessfully: protocol.ProtocolVersion,
		ServerHost:       host,
		ServerPort:       int32(portNum),
	}
	if err := protocol.WritePacket(conn, ping); err != nil {
		fmt.Printf("write ping failed: %v\n", err)
		return
	}

	resp, err := protocol.ReadPacket(conn, protocol.DirectionClientbound)
	if err != nil {
		fmt.Printf("read response failed: %v\n", err)
		return
	}

	disconnect, ok := resp.(*protocol.Packet255KickDisconnect)
	if !ok {
		fmt.Printf("unexpected response type: %T\n", resp)
		return
	}

	reason := disconnect.Reason
	fmt.Printf("raw status: %q\n", reason)

	parts := strings.Split(reason, "\x00")
	if len(parts) == 6 && strings.HasPrefix(parts[0], "\u00a71") {
		fmt.Printf("protocol=%s version=%s motd=%s players=%s/%s\n", parts[1], parts[2], parts[3], parts[4], parts[5])
	}
}

func runLoginProbe(addr, username string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("connect failed: %v\n", err)
		return
	}
	defer conn.Close()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("invalid addr %q: %v\n", addr, err)
		return
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		fmt.Printf("invalid port %q: %v\n", port, err)
		return
	}

	if err := protocol.WritePacket(conn, &protocol.Packet2ClientProtocol{
		ProtocolVersion: protocol.ProtocolVersion,
		Username:        username,
		ServerHost:      host,
		ServerPort:      int32(portNum),
	}); err != nil {
		fmt.Printf("write Packet2 failed: %v\n", err)
		return
	}

	first, err := protocol.ReadPacket(conn, protocol.DirectionClientbound)
	if err != nil {
		fmt.Printf("read first login packet failed: %v\n", err)
		return
	}

	auth, ok := first.(*protocol.Packet253ServerAuthData)
	if !ok {
		if kick, isKick := first.(*protocol.Packet255KickDisconnect); isKick {
			fmt.Printf("kicked: %s\n", kick.Reason)
			return
		}
		fmt.Printf("unexpected first packet type: %T\n", first)
		return
	}

	pub, err := crypt.DecodePublicKey(auth.PublicKey)
	if err != nil {
		fmt.Printf("decode server public key failed: %v\n", err)
		return
	}
	sharedKey, err := crypt.CreateNewSharedKey()
	if err != nil {
		fmt.Printf("create shared key failed: %v\n", err)
		return
	}
	encKey, err := crypt.EncryptData(pub, sharedKey)
	if err != nil {
		fmt.Printf("encrypt shared key failed: %v\n", err)
		return
	}
	encToken, err := crypt.EncryptData(pub, auth.VerifyToken)
	if err != nil {
		fmt.Printf("encrypt verify token failed: %v\n", err)
		return
	}

	if err := protocol.WritePacket(conn, &protocol.Packet252SharedKey{
		SharedSecret: encKey,
		VerifyToken:  encToken,
	}); err != nil {
		fmt.Printf("write Packet252 failed: %v\n", err)
		return
	}

	encryptedOut, err := crypt.EncryptOutputStream(sharedKey, conn)
	if err != nil {
		fmt.Printf("enable output encryption failed: %v\n", err)
		return
	}

	ack, err := protocol.ReadPacket(conn, protocol.DirectionClientbound)
	if err != nil {
		fmt.Printf("read Packet252 ack failed: %v\n", err)
		return
	}
	if _, ok := ack.(*protocol.Packet252SharedKey); !ok {
		fmt.Printf("unexpected packet after Packet252: %T\n", ack)
		return
	}

	decryptedIn, err := crypt.DecryptInputStream(sharedKey, conn)
	if err != nil {
		fmt.Printf("enable input decryption failed: %v\n", err)
		return
	}

	if err := protocol.WritePacket(encryptedOut, &protocol.Packet205ClientCommand{ForceRespawn: 0}); err != nil {
		fmt.Printf("write Packet205 failed: %v\n", err)
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	fmt.Println("login init packets:")
	for {
		packet, err := protocol.ReadPacket(decryptedIn, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		fmt.Printf(" - id=%d type=%T\n", packet.PacketID(), packet)
	}
}

func runPlayGUI(session *netclient.Session, cfg rendergui.Config) {
	if session == nil {
		fmt.Println("gui run failed: session is nil")
		return
	}
	defer func() {
		_ = session.Close()
		_ = session.Wait()
	}()
	snap := session.Snapshot()
	username := snap.Username
	if username == "" {
		username = "Player"
	}
	fmt.Printf("connected as %s\n", username)
	fmt.Println("GUI controls: main menu mouse click; ingame WASD move, double-Space toggle fly (creative), Space up (flying), Shift sneak/down (flying), Ctrl sprint, mouse look, 1-9 hotbar, E inventory, T or Enter chat, / opens command chat, Up/Down chat history, left click dig/swing, right click place, F1 hide HUD, F3 debug, Esc close UI/pause.")

	if err := rendergui.Run(session, cfg); err != nil {
		fmt.Printf("gui run failed: %v\n", err)
	}
}

func runPlayCLI(addr, username string) {
	session, err := netclient.DialAndLogin(addr, username)
	if err != nil {
		fmt.Printf("connect/login failed: %v\n", err)
		return
	}
	defer session.Close()

	fmt.Printf("connected to %s as %s\n", addr, username)
	printPlayHelp()

	go func() {
		for event := range session.Events() {
			switch event.Type {
			case netclient.EventChat:
				fmt.Printf("[chat] %s\n", event.Message)
			case netclient.EventKick:
				fmt.Printf("[kick] %s\n", event.Message)
			case netclient.EventDisconnect:
				fmt.Printf("[disconnect] %s\n", event.Message)
			default:
				fmt.Printf("[info] %s\n", event.Message)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-session.Done():
				return
			case <-ticker.C:
				_ = session.SendOnGround(true)
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024), 64*1024)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !handlePlayCommand(session, line) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("stdin read failed: %v\n", err)
	}

	_ = session.Close()
	if err := session.Wait(); err != nil && !errors.Is(err, net.ErrClosed) {
		fmt.Printf("session ended with error: %v\n", err)
	}
}

func printPlayHelp() {
	fmt.Println("commands:")
	fmt.Println("  help")
	fmt.Println("  where")
	fmt.Println("  entities")
	fmt.Println("  move <dx> <dy> <dz>")
	fmt.Println("  look <yaw> <pitch>")
	fmt.Println("  w|a|s|d [distance]")
	fmt.Println("  swing")
	fmt.Println("  use <entityID> [interact|attack]")
	fmt.Println("  sneak <on|off>")
	fmt.Println("  sprint <on|off>")
	fmt.Println("  slot <0-8>")
	fmt.Println("  click <slot|-999> [left|right] [normal|shift]")
	fmt.Println("  closewin")
	fmt.Println("  dig <x> <y> <z> [face]")
	fmt.Println("  place <x> <y> <z> <face> [itemID] [meta]")
	fmt.Println("  block <x> <y> <z>")
	fmt.Println("  say <message>")
	fmt.Println("  cmd <rawCommandWithoutSlash>")
	fmt.Println("  /<serverCommand>")
	fmt.Println("  tip: use /give <id> [count] [damage] to populate inventory")
	fmt.Println("  quit")
}

func handlePlayCommand(session *netclient.Session, line string) bool {
	if strings.HasPrefix(line, "/") {
		if err := session.SendChat(line); err != nil {
			fmt.Printf("send command failed: %v\n", err)
		}
		return true
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return true
	}

	cmd := strings.ToLower(fields[0])
	switch cmd {
	case "help":
		printPlayHelp()
	case "quit", "exit":
		return false
	case "where":
		snap := session.Snapshot()
		fmt.Printf(
			"pos=(%.3f, %.3f, %.3f) yaw=%.2f pitch=%.2f hp=%.1f food=%d gm=%d fly=%t/%t creative=%t held=%d(id=%d,count=%d,dmg=%d) chunks=%d entities=%d time=%d/%d\n",
			snap.PlayerX,
			snap.PlayerY,
			snap.PlayerZ,
			snap.PlayerYaw,
			snap.PlayerPitch,
			snap.Health,
			snap.Food,
			snap.GameType,
			snap.CanFly,
			snap.IsFlying,
			snap.IsCreative,
			snap.HeldSlot,
			snap.HeldItemID,
			snap.HeldCount,
			snap.HeldDamage,
			snap.LoadedChunks,
			snap.TrackedEntities,
			snap.WorldAge,
			snap.WorldTime,
		)
	case "entities":
		entities := session.EntitiesSnapshot()
		if len(entities) == 0 {
			fmt.Println("no tracked entities")
			return true
		}
		for _, ent := range entities {
			fmt.Printf("id=%d type=%d name=%q pos=(%.3f, %.3f, %.3f) yaw=%d pitch=%d head=%d sneak=%t sprint=%t use=%t\n", ent.EntityID, ent.Type, ent.Name, ent.X, ent.Y, ent.Z, ent.Yaw, ent.Pitch, ent.HeadYaw, ent.Sneaking, ent.Sprinting, ent.UsingItem)
		}
	case "move":
		if len(fields) != 4 {
			fmt.Println("usage: move <dx> <dy> <dz>")
			return true
		}
		dx, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			fmt.Printf("invalid dx: %v\n", err)
			return true
		}
		dy, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			fmt.Printf("invalid dy: %v\n", err)
			return true
		}
		dz, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			fmt.Printf("invalid dz: %v\n", err)
			return true
		}
		if err := session.MoveRelative(dx, dy, dz); err != nil {
			fmt.Printf("move failed: %v\n", err)
		}
	case "look":
		if len(fields) != 3 {
			fmt.Println("usage: look <yaw> <pitch>")
			return true
		}
		yaw, err := strconv.ParseFloat(fields[1], 32)
		if err != nil {
			fmt.Printf("invalid yaw: %v\n", err)
			return true
		}
		pitch, err := strconv.ParseFloat(fields[2], 32)
		if err != nil {
			fmt.Printf("invalid pitch: %v\n", err)
			return true
		}
		if err := session.Look(float32(yaw), float32(pitch)); err != nil {
			fmt.Printf("look failed: %v\n", err)
		}
	case "w", "a", "s", "d":
		dist := 1.0
		if len(fields) == 2 {
			v, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				fmt.Printf("invalid distance: %v\n", err)
				return true
			}
			dist = v
		}
		if len(fields) > 2 {
			fmt.Printf("usage: %s [distance]\n", cmd)
			return true
		}
		forward := 0.0
		strafe := 0.0
		switch cmd {
		case "w":
			forward = 1.0
		case "s":
			forward = -1.0
		case "a":
			strafe = -1.0
		case "d":
			strafe = 1.0
		}
		snap := session.Snapshot()
		yawRad := float64(snap.PlayerYaw) * math.Pi / 180.0
		dx := (-math.Sin(yawRad)*forward + math.Cos(yawRad)*strafe) * dist
		dz := (math.Cos(yawRad)*forward + math.Sin(yawRad)*strafe) * dist
		if err := session.MoveRelative(dx, 0, dz); err != nil {
			fmt.Printf("move failed: %v\n", err)
		}
	case "swing":
		if err := session.SwingArm(); err != nil {
			fmt.Printf("swing failed: %v\n", err)
		}
	case "use":
		if len(fields) < 2 || len(fields) > 3 {
			fmt.Println("usage: use <entityID> [interact|attack]")
			return true
		}
		entityID, err := strconv.Atoi(fields[1])
		if err != nil {
			fmt.Printf("invalid entityID: %v\n", err)
			return true
		}
		attack := false
		if len(fields) == 3 {
			switch strings.ToLower(fields[2]) {
			case "interact":
				attack = false
			case "attack":
				attack = true
			default:
				fmt.Println("mode must be interact or attack")
				return true
			}
		}
		if err := session.UseEntity(int32(entityID), attack); err != nil {
			fmt.Printf("use failed: %v\n", err)
		}
	case "sneak":
		if len(fields) != 2 {
			fmt.Println("usage: sneak <on|off>")
			return true
		}
		enabled, ok := parseToggle(fields[1])
		if !ok {
			fmt.Println("usage: sneak <on|off>")
			return true
		}
		if err := session.SetSneaking(enabled); err != nil {
			fmt.Printf("sneak failed: %v\n", err)
		}
	case "sprint":
		if len(fields) != 2 {
			fmt.Println("usage: sprint <on|off>")
			return true
		}
		enabled, ok := parseToggle(fields[1])
		if !ok {
			fmt.Println("usage: sprint <on|off>")
			return true
		}
		if err := session.SetSprinting(enabled); err != nil {
			fmt.Printf("sprint failed: %v\n", err)
		}
	case "slot":
		if len(fields) != 2 {
			fmt.Println("usage: slot <0-8>")
			return true
		}
		slot, err := strconv.Atoi(fields[1])
		if err != nil || slot < 0 || slot > 8 {
			fmt.Println("slot must be in range 0-8")
			return true
		}
		if err := session.SelectHotbar(int16(slot)); err != nil {
			fmt.Printf("slot switch failed: %v\n", err)
		}
	case "click":
		if len(fields) < 2 || len(fields) > 4 {
			fmt.Println("usage: click <slot|-999> [left|right] [normal|shift]")
			return true
		}
		slot, err := strconv.Atoi(fields[1])
		if err != nil {
			fmt.Printf("invalid slot: %v\n", err)
			return true
		}
		right := false
		if len(fields) >= 3 {
			switch strings.ToLower(fields[2]) {
			case "left":
				right = false
			case "right":
				right = true
			default:
				fmt.Println("button must be left or right")
				return true
			}
		}
		shift := false
		if len(fields) == 4 {
			switch strings.ToLower(fields[3]) {
			case "normal":
				shift = false
			case "shift":
				shift = true
			default:
				fmt.Println("mode must be normal or shift")
				return true
			}
		}
		if err := session.ClickWindowSlot(int16(slot), right, shift); err != nil {
			fmt.Printf("click failed: %v\n", err)
		}
	case "closewin":
		if err := session.CloseInventoryWindow(); err != nil {
			fmt.Printf("close window failed: %v\n", err)
		}
	case "dig":
		if len(fields) < 4 || len(fields) > 5 {
			fmt.Println("usage: dig <x> <y> <z> [face]")
			return true
		}
		x, err := strconv.Atoi(fields[1])
		if err != nil {
			fmt.Printf("invalid x: %v\n", err)
			return true
		}
		y, err := strconv.Atoi(fields[2])
		if err != nil {
			fmt.Printf("invalid y: %v\n", err)
			return true
		}
		z, err := strconv.Atoi(fields[3])
		if err != nil {
			fmt.Printf("invalid z: %v\n", err)
			return true
		}
		face := 1
		if len(fields) == 5 {
			face, err = strconv.Atoi(fields[4])
			if err != nil {
				fmt.Printf("invalid face: %v\n", err)
				return true
			}
		}
		if err := session.DigBlock(int32(x), int32(y), int32(z), int32(face)); err != nil {
			fmt.Printf("dig failed: %v\n", err)
		}
	case "place":
		if len(fields) < 5 || len(fields) > 7 {
			fmt.Println("usage: place <x> <y> <z> <face> [itemID] [meta]")
			return true
		}
		x, err := strconv.Atoi(fields[1])
		if err != nil {
			fmt.Printf("invalid x: %v\n", err)
			return true
		}
		y, err := strconv.Atoi(fields[2])
		if err != nil {
			fmt.Printf("invalid y: %v\n", err)
			return true
		}
		z, err := strconv.Atoi(fields[3])
		if err != nil {
			fmt.Printf("invalid z: %v\n", err)
			return true
		}
		face, err := strconv.Atoi(fields[4])
		if err != nil {
			fmt.Printf("invalid face: %v\n", err)
			return true
		}
		if len(fields) == 5 {
			if err := session.PlaceHeldBlock(int32(x), int32(y), int32(z), int32(face)); err != nil {
				fmt.Printf("place failed: %v\n", err)
			}
			return true
		}

		itemID, err := strconv.Atoi(fields[5])
		if err != nil {
			fmt.Printf("invalid itemID: %v\n", err)
			return true
		}
		meta := 0
		if len(fields) == 7 {
			meta, err = strconv.Atoi(fields[6])
			if err != nil {
				fmt.Printf("invalid meta: %v\n", err)
				return true
			}
		}
		if err := session.PlaceBlock(int32(x), int32(y), int32(z), int32(face), int16(itemID), int16(meta)); err != nil {
			fmt.Printf("place failed: %v\n", err)
		}
	case "block":
		if len(fields) != 4 {
			fmt.Println("usage: block <x> <y> <z>")
			return true
		}
		x, err := strconv.Atoi(fields[1])
		if err != nil {
			fmt.Printf("invalid x: %v\n", err)
			return true
		}
		y, err := strconv.Atoi(fields[2])
		if err != nil {
			fmt.Printf("invalid y: %v\n", err)
			return true
		}
		z, err := strconv.Atoi(fields[3])
		if err != nil {
			fmt.Printf("invalid z: %v\n", err)
			return true
		}
		id, meta, ok := session.BlockAt(x, y, z)
		if !ok {
			fmt.Println("block cache miss (chunk not loaded yet)")
			return true
		}
		fmt.Printf("block (%d,%d,%d) -> id=%d meta=%d\n", x, y, z, id, meta)
	case "say":
		if len(fields) < 2 {
			fmt.Println("usage: say <message>")
			return true
		}
		msg := strings.TrimSpace(line[len(fields[0]):])
		if err := session.SendChat(msg); err != nil {
			fmt.Printf("chat failed: %v\n", err)
		}
	case "cmd":
		if len(fields) < 2 {
			fmt.Println("usage: cmd <rawCommandWithoutSlash>")
			return true
		}
		raw := strings.TrimSpace(line[len(fields[0]):])
		if !strings.HasPrefix(raw, "/") {
			raw = "/" + raw
		}
		if err := session.SendChat(raw); err != nil {
			fmt.Printf("command failed: %v\n", err)
		}
	default:
		fmt.Printf("unknown command: %s\n", cmd)
	}

	return true
}

func parseToggle(v string) (bool, bool) {
	switch strings.ToLower(v) {
	case "on", "1", "true", "start":
		return true, true
	case "off", "0", "false", "stop":
		return false, true
	default:
		return false, false
	}
}
