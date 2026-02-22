//go:build cgo

package gui

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lulaide/gomc/pkg/audio"
	netclient "github.com/lulaide/gomc/pkg/network/client"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const (
	playerEyeHeight = 1.6200000047683716
	playerHeight    = 1.8
	playerHalfWidth = 0.3
	interactReach   = 5.0
	raycastStep     = 0.05
	jumpVelocity    = 0.42
	gravityPerTick  = 0.08
	dragPerTick     = 0.98
	moveTickSeconds = 0.05
	maxMoveCatchUp  = 4
	maxChunkBuilds  = 4
	defaultFPSMode  = 1
	sprintTapTicks  = 7
)

var (
	// Translation reference:
	// - net.minecraft.src.GameSettings.RENDER_DISTANCES
	renderDistanceModeNames = []string{"Far", "Normal", "Short", "Tiny"}
	// Translation reference:
	// - net.minecraft.src.GameSettings.LIMIT_FRAMERATES
	framerateModeNames = []string{"Max FPS", "Balanced", "Power saver"}
	// Translation reference:
	// - net.minecraft.src.GameSettings.GUISCALES
	guiScaleModeNames = []string{"Auto", "Small", "Normal", "Large"}
)

const (
	buttonIDPauseDisconnect   = 1
	buttonIDPauseReturnToGame = 4
	buttonIDPauseAchievements = 5
	buttonIDPauseStats        = 6
	buttonIDPauseOptions      = 0
	buttonIDPauseShareToLAN   = 7
	buttonIDMenuSingleplayer  = 1
	buttonIDMenuMultiplayer   = 2
	buttonIDMenuOnline        = 14
	buttonIDMenuOptions       = 0
	buttonIDMenuQuit          = 4
	buttonIDMenuLanguage      = 5
)

type pauseScreen int

const (
	pauseScreenMain pauseScreen = iota
	pauseScreenOptions
	pauseScreenVideo
	pauseScreenControls
	pauseScreenKeyBindings
	pauseScreenSounds
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

type blockTarget struct {
	X    int
	Y    int
	Z    int
	Face int32
	Hit  bool
	Dist float64
	MinX float64
	MinY float64
	MinZ float64
	MaxX float64
	MaxY float64
	MaxZ float64
}

type entityTarget struct {
	EntityID int32
	Dist     float64
	Hit      bool
}

type visibleFaces struct {
	Down  bool
	Up    bool
	North bool
	South bool
	West  bool
	East  bool
}

func (f visibleFaces) any() bool {
	return f.Down || f.Up || f.North || f.South || f.West || f.East
}

type App struct {
	session *netclient.Session
	window  *glfw.Window

	width        int
	height       int
	guiW         int
	guiH         int
	guiS         int
	guiScaleMode int

	renderDistance int
	moveSpeed      float64
	mouseSens      float64
	musicVolume    float64
	soundVolume    float64
	invertMouse    bool
	fovSetting     float64
	viewBobbing    bool
	fancyGraphics  bool
	cloudsEnabled  bool
	activeWorld    string
	playWorldFn    func(worldDir string) (*netclient.Session, error)

	yaw   float64
	pitch float64
	velY  float64

	walkDistance     float64
	prevWalkDistance float64
	cameraYaw        float64
	prevCameraYaw    float64
	cameraPitch      float64
	prevCameraPitch  float64

	firstMouse bool
	lastMouseX float64
	lastMouseY float64
	mouseX     int
	mouseY     int

	prevLeftMouse   bool
	prevRightMouse  bool
	prevMiddleMouse bool
	prevF1          bool
	prevF3          bool
	prevEsc         bool
	prevEnter       bool
	prevSpace       bool
	prevBackspace   bool
	prevE           bool
	prevT           bool
	prevSlash       bool
	prevUp          bool
	prevDown        bool
	prevSneakKey    bool
	prevAttackInput bool
	prevUseInput    bool
	prevDropInput   bool
	prevPickInput   bool
	prevDigit       [9]bool
	prevForwardKey  bool

	hudHidden      bool
	showDebug      bool
	showPlayerList bool
	paused         bool
	pauseScreen    pauseScreen
	mainMenu       bool
	menuScreen     menuScreen
	inventoryOpen  bool
	chatInputOpen  bool
	chatInput      string
	chatDraft      string
	chatHistory    []string
	chatHistPos    int

	pauseButtons       []*guiButton
	pauseOptionButtons []*guiButton
	mainButtons        []*guiButton
	singleButtons      []*guiButton
	multiButtons       []*guiButton
	optionButtons      []*guiButton
	videoButtons       []*guiButton
	controlButtons     []*guiButton
	keyBindButtons     []*guiButton
	soundButtons       []*guiButton
	createButtons      []*guiButton
	renameButtons      []*guiButton
	singleWorlds       []string
	singleWorldMeta    map[string]singleWorldMeta
	selectedWorld      int
	menuStatus         string
	optionDifficulty   int
	createWorldName    string
	createWorldSeed    string
	createWorldMode    int
	createMapFeature   bool
	createAllowCheats  bool
	createCheatsTog    bool
	createBonusChest   bool
	createWorldTypeID  int
	createGeneratorOp  string
	createMoreOptions  bool
	createFolderName   string
	recreateSource     string
	renameWorldDir     string
	renameWorldName    string
	activeTextField    menuTextField
	typedRuneQueue     []rune
	keyPressQueue      []glfw.Key

	eventsCh   <-chan netclient.Event
	chatMu     sync.Mutex
	chatLines  []chatLine
	chatClosed bool

	assetsRoot     string
	optionsPath    string
	optionsKV      map[string]string
	keyBindings    []keyBindingConfig
	keyBindCapture int
	texWidgets     *texture2D
	texIcons       *texture2D
	texOptionsBG   *texture2D
	texInventory   *texture2D
	texTitle       *texture2D
	texPanorama    [6]*texture2D
	texMenuView    *texture2D
	texSun         *texture2D
	texMoonPhases  *texture2D
	texClouds      *texture2D
	font           *fontRenderer
	splashText     string
	panoramaTick   int
	panoramaFrac   float64
	lastSpaceTap   time.Time
	sprintTimer    int
	localSprinting bool
	lastOnGround   time.Time
	localOnGround  bool
	prevTab        bool
	prevBacksp     bool

	blockTextureDefs   map[int]blockTextureDef
	blockTextures      map[string]*texture2D
	grassColorMap      []uint32
	foliageColorMap    []uint32
	entityTextures     map[string]*texture2D
	chunkRenderCache   map[chunk.CoordIntPair]*chunkRenderEntry
	renderFrame        uint64
	currentFPS         int
	fpsWindowStart     time.Time
	fpsFrames          int
	limitFramerateMode int
	moveTickAccum      float64
	skyStarListID      uint32
	skyListID          uint32
	skyList2ID         uint32

	// Client-side interpolation track for camera smoothness between 20 TPS movement steps.
	renderTrackInit    bool
	renderPrevX        float64
	renderPrevY        float64
	renderPrevZ        float64
	renderCurrX        float64
	renderCurrY        float64
	renderCurrZ        float64
	renderLerpTime     float64
	renderArmYaw       float64
	renderArmPitch     float64
	prevRenderArmYaw   float64
	prevRenderArmPitch float64
	handSwingStart     time.Time
}

type chatLine struct {
	Message string
	AddedAt time.Time
	Color   int
}

type chunkRenderEntry struct {
	listOpaque      uint32
	listTranslucent uint32
	revision        uint64
	lastUsedFrame   uint64
}

type textureBatch struct {
	tex   *texture2D
	verts []texturedVertex
}

type texturedVertex struct {
	x float32
	y float32
	z float32
	u float32
	v float32
	r float32
	g float32
	b float32
}

func Run(session *netclient.Session, cfg Config) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if cfg.Width <= 0 {
		cfg.Width = 1280
	}
	if cfg.Height <= 0 {
		cfg.Height = 720
	}
	if cfg.RenderDistance <= 0 {
		cfg.RenderDistance = 10
	}
	if cfg.MouseSensitivity < 0 {
		cfg.MouseSensitivity = 0.5
	}
	if cfg.MouseSensitivity > 1.0 {
		cfg.MouseSensitivity = 1.0
	}
	if cfg.MoveSpeed <= 0 {
		cfg.MoveSpeed = 4.3
	}
	if cfg.FPSLimitMode < 0 || cfg.FPSLimitMode > 2 {
		cfg.FPSLimitMode = defaultFPSMode
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := glfw.Init(); err != nil {
		return fmt.Errorf("glfw init failed: %w", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.Resizable, glfw.True)

	window, err := glfw.CreateWindow(cfg.Width, cfg.Height, "GoMC GUI", nil, nil)
	if err != nil {
		return fmt.Errorf("create window failed: %w", err)
	}
	defer window.Destroy()

	window.MakeContextCurrent()
	// Keep startup uncoupled from monitor refresh; options screen will own VSync toggle.
	glfw.SwapInterval(0)
	if err := gl.Init(); err != nil {
		return fmt.Errorf("gl init failed: %w", err)
	}

	app := &App{
		session:            session,
		window:             window,
		width:              cfg.Width,
		height:             cfg.Height,
		guiScaleMode:       0,
		renderDistance:     renderDistanceChunksToMode(cfg.RenderDistance),
		moveSpeed:          cfg.MoveSpeed,
		mouseSens:          cfg.MouseSensitivity,
		musicVolume:        1.0,
		soundVolume:        1.0,
		invertMouse:        false,
		fovSetting:         0.0,
		viewBobbing:        true,
		fancyGraphics:      true,
		cloudsEnabled:      true,
		activeWorld:        cfg.CurrentWorld,
		playWorldFn:        cfg.PlayWorld,
		firstMouse:         true,
		lastMouseX:         float64(cfg.Width) / 2,
		lastMouseY:         float64(cfg.Height) / 2,
		assetsRoot:         discoverAssetsRoot(),
		optionsKV:          make(map[string]string),
		eventsCh:           session.Events(),
		mainMenu:           cfg.StartInMainMenu,
		pauseScreen:        pauseScreenMain,
		menuScreen:         menuScreenMain,
		keyBindCapture:     -1,
		selectedWorld:      -1,
		singleWorldMeta:    make(map[string]singleWorldMeta),
		optionDifficulty:   1,
		createMapFeature:   true,
		createWorldTypeID:  0,
		chunkRenderCache:   make(map[chunk.CoordIntPair]*chunkRenderEntry),
		limitFramerateMode: cfg.FPSLimitMode,
	}
	app.initDefaultKeyBindings()
	app.optionsPath = discoverOptionsPath(app.assetsRoot)
	app.loadOptionsFile()
	audio.InitWithAssets(app.assetsRoot)
	defer func() {
		if app.session != nil {
			_ = app.session.Close()
			_ = app.session.Wait()
		}
	}()
	fbW, fbH := window.GetFramebufferSize()
	if fbW > 0 {
		app.width = fbW
	}
	if fbH > 0 {
		app.height = fbH
	}
	if exePath, err := os.Executable(); err == nil {
		fmt.Printf("gui startup: exe=%s assets=%s mainMenu=%t\n", exePath, app.assetsRoot, app.mainMenu)
	} else {
		fmt.Printf("gui startup: assets=%s mainMenu=%t\n", app.assetsRoot, app.mainMenu)
	}

	if err := app.loadAssets(); err != nil {
		fmt.Printf("gui asset warning: %v\n", err)
	}
	panoramaLoaded := true
	for i := range app.texPanorama {
		if app.texPanorama[i] == nil {
			panoramaLoaded = false
			break
		}
	}
	fmt.Printf("gui assets: font=%t widgets=%t icons=%t options_bg=%t inventory=%t title=%t panorama=%t skybox_rt=%t sun=%t moon=%t clouds=%t\n",
		app.font != nil, app.texWidgets != nil, app.texIcons != nil, app.texOptionsBG != nil, app.texInventory != nil, app.texTitle != nil, panoramaLoaded, app.texMenuView != nil, app.texSun != nil, app.texMoonPhases != nil, app.texClouds != nil)
	defer app.releaseAssets()
	app.updateGUIMetrics()
	winW, winH := window.GetSize()
	scaleX, scaleY := window.GetContentScale()
	fmt.Printf("gui metrics: window=%dx%d framebuffer=%dx%d guiScale=%d gui=%dx%d contentScale=%.2fx%.2f\n",
		winW, winH, app.width, app.height, app.uiScale(), app.uiWidth(), app.uiHeight(), scaleX, scaleY)
	app.initAllMenuButtons()

	app.applyCursorMode()
	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		app.width = maxInt(width, 1)
		app.height = maxInt(height, 1)
		app.updateGUIMetrics()
		gl.Viewport(0, 0, int32(app.width), int32(app.height))
	})
	window.SetCharCallback(func(_ *glfw.Window, char rune) {
		app.enqueueTypedRune(char)
	})
	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Press {
			app.enqueueKeyPress(key)
		}
	})
	gl.Viewport(0, 0, int32(app.width), int32(app.height))

	app.initGLState()
	return app.loop()
}

func (a *App) loadAssets() error {
	var errs []string

	loadGUI := func(rel string, nearest bool) *texture2D {
		tex, _, err := loadTexture2DWithFlip(filepath.Join(a.assetsRoot, rel), nearest, false)
		if err != nil {
			errs = append(errs, err.Error())
			return nil
		}
		return tex
	}

	a.texWidgets = loadGUI(filepath.Join("textures", "gui", "widgets.png"), true)
	a.texIcons = loadGUI(filepath.Join("textures", "gui", "icons.png"), true)
	a.texOptionsBG = loadGUI(filepath.Join("textures", "gui", "options_background.png"), true)
	a.texInventory = loadGUI(filepath.Join("textures", "gui", "container", "inventory.png"), true)
	a.texTitle = loadGUI(filepath.Join("textures", "gui", "title", "minecraft.png"), true)
	for i := 0; i < len(a.texPanorama); i++ {
		a.texPanorama[i] = loadGUI(filepath.Join("textures", "gui", "title", "background", fmt.Sprintf("panorama_%d.png", i)), true)
	}
	a.texSun = loadGUI(filepath.Join("textures", "environment", "sun.png"), false)
	a.texMoonPhases = loadGUI(filepath.Join("textures", "environment", "moon_phases.png"), false)
	a.texClouds = loadGUI(filepath.Join("textures", "environment", "clouds.png"), true)
	if a.texClouds != nil {
		a.texClouds.setWrapRepeat(true)
	}
	menuView, err := newEmptyTexture2D(256, 256, false)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		a.texMenuView = menuView
	}

	font, err := loadFontRenderer(
		filepath.Join(a.assetsRoot, "textures", "font", "ascii.png"),
		discoverFontCharsPath(a.assetsRoot),
	)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		a.font = font
	}

	a.splashText = a.loadSplashText(filepath.Join(a.assetsRoot, "texts", "splashes.txt"))
	if a.splashText == "" {
		a.splashText = "missingno"
	}
	if err := a.loadBlockTextures(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := a.loadEntityTextures(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (a *App) loadSplashText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	splashes := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		splashes = append(splashes, line)
	}
	if len(splashes) == 0 {
		return ""
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return splashes[rng.Intn(len(splashes))]
}

func (a *App) releaseAssets() {
	a.releaseChunkRenderCache()
	a.releaseSkyRenderLists()
	if a.font != nil {
		a.font.delete()
	}
	if a.texWidgets != nil {
		a.texWidgets.delete()
	}
	if a.texIcons != nil {
		a.texIcons.delete()
	}
	if a.texOptionsBG != nil {
		a.texOptionsBG.delete()
	}
	if a.texInventory != nil {
		a.texInventory.delete()
	}
	if a.texTitle != nil {
		a.texTitle.delete()
	}
	for i := range a.texPanorama {
		if a.texPanorama[i] != nil {
			a.texPanorama[i].delete()
		}
	}
	if a.texMenuView != nil {
		a.texMenuView.delete()
	}
	if a.texSun != nil {
		a.texSun.delete()
	}
	if a.texMoonPhases != nil {
		a.texMoonPhases.delete()
	}
	if a.texClouds != nil {
		a.texClouds.delete()
	}
	for name, tex := range a.blockTextures {
		if tex != nil {
			tex.delete()
		}
		delete(a.blockTextures, name)
	}
	for name, tex := range a.entityTextures {
		if tex != nil {
			tex.delete()
		}
		delete(a.entityTextures, name)
	}
}

func (a *App) releaseChunkRenderCache() {
	for key, entry := range a.chunkRenderCache {
		if entry != nil {
			if entry.listOpaque != 0 {
				gl.DeleteLists(entry.listOpaque, 1)
			}
			if entry.listTranslucent != 0 {
				gl.DeleteLists(entry.listTranslucent, 1)
			}
		}
		delete(a.chunkRenderCache, key)
	}
}

func (a *App) releaseSkyRenderLists() {
	if a.skyStarListID != 0 {
		gl.DeleteLists(a.skyStarListID, 3)
	}
	a.skyStarListID = 0
	a.skyListID = 0
	a.skyList2ID = 0
}

func (a *App) initGLState() {
	gl.ClearColor(0.53, 0.81, 0.92, 1.0)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	// Vanilla-like cutout rendering for alpha textures (flowers/grass/reeds).
	gl.Enable(gl.ALPHA_TEST)
	gl.AlphaFunc(gl.GREATER, 0.1)
	gl.Disable(gl.BLEND)
	a.initSkyRenderLists()
}

func autoGUIScale(displayW, displayH, guiScaleMode int) int {
	scale := 1
	mode := clampInt(guiScaleMode, 0, len(guiScaleModeNames)-1)
	maxScale := 1000
	if mode > 0 {
		maxScale = mode
	}
	for scale < maxScale && displayW/(scale+1) >= 320 && displayH/(scale+1) >= 240 {
		scale++
	}
	return maxInt(scale, 1)
}

func (a *App) uiWidth() int {
	if a.guiW > 0 {
		return a.guiW
	}
	return maxInt(a.width, 1)
}

func (a *App) uiHeight() int {
	if a.guiH > 0 {
		return a.guiH
	}
	return maxInt(a.height, 1)
}

func (a *App) uiScale() int {
	if a.guiS > 0 {
		return a.guiS
	}
	return 1
}

func (a *App) updateGUIMetrics() {
	newS := autoGUIScale(maxInt(a.width, 1), maxInt(a.height, 1), a.guiScaleMode)
	// Translation reference:
	// - net.minecraft.src.ScaledResolution
	// Vanilla uses ceil(display / scaleFactor).
	newW := maxInt(int(math.Ceil(float64(a.width)/float64(newS))), 1)
	newH := maxInt(int(math.Ceil(float64(a.height)/float64(newS))), 1)
	changed := newS != a.guiS || newW != a.guiW || newH != a.guiH
	a.guiS = newS
	a.guiW = newW
	a.guiH = newH

	if !changed {
		return
	}
	if a.paused {
		a.initPauseButtons()
		a.initPauseOptionsButtons()
	}
	if a.mainMenu {
		a.initAllMenuButtons()
	}
}

func (a *App) loop() error {
	var snap netclient.StateSnapshot
	if a.session != nil {
		snap = a.session.Snapshot()
		a.yaw = float64(snap.PlayerYaw)
		a.pitch = float64(snap.PlayerPitch)
		a.renderArmYaw = a.yaw
		a.renderArmPitch = a.pitch
		a.prevRenderArmYaw = a.yaw
		a.prevRenderArmPitch = a.pitch
		a.localOnGround = snap.OnGround
		a.resetRenderTrackFromSnapshot(snap)
	}

	lastTime := time.Now()
	lastTitleUpdate := time.Now()
	titleInterval := 250 * time.Millisecond

	for !a.window.ShouldClose() {
		if a.session != nil {
			select {
			case <-a.session.Done():
				err := a.session.Wait()
				a.session = nil
				a.eventsCh = nil
				a.chatClosed = true
				a.paused = false
				a.pauseScreen = pauseScreenMain
				a.inventoryOpen = false
				a.chatInputOpen = false
				a.chatInput = ""
				a.chatDraft = ""
				a.chatHistPos = len(a.chatHistory)
				a.mainMenu = true
				a.menuScreen = menuScreenMain
				if err != nil {
					a.menuStatus = fmt.Sprintf("Disconnected: %v", err)
				}
				a.initAllMenuButtons()
				a.applyCursorMode()
			default:
			}
		}

		now := time.Now()
		delta := now.Sub(lastTime).Seconds()
		if delta <= 0 {
			delta = 1.0 / 60.0
		}
		if delta > 0.1 {
			delta = 0.1
		}
		lastTime = now

		glfw.PollEvents()
		if !a.handleInput(delta) {
			return nil
		}
		if a.session != nil && now.Sub(a.lastOnGround) >= time.Second {
			_ = a.session.SendOnGround(a.localOnGround)
			a.lastOnGround = now
		}
		a.drainSessionEvents()
		if a.mainMenu && a.menuScreen == menuScreenMain {
			a.panoramaFrac += delta * 20.0
			whole := int(a.panoramaFrac)
			if whole > 0 {
				a.panoramaTick += whole
				a.panoramaFrac -= float64(whole)
			}
		} else {
			a.panoramaFrac = 0
		}
		a.advanceAnimatedTextures(now)

		target := blockTarget{}
		if a.session != nil {
			snap = a.session.Snapshot()
			if a.syncRenderTrackFromSnapshot(snap) {
				a.renderLerpTime = 0
			} else {
				a.renderLerpTime += delta
				if a.renderLerpTime > moveTickSeconds {
					a.renderLerpTime = moveTickSeconds
				}
			}
			if !a.mainMenu {
				target = a.pickBlockTarget(snap, interactReach)
			}
		}
		a.prevRenderArmPitch = a.renderArmPitch
		a.prevRenderArmYaw = a.renderArmYaw
		a.renderArmPitch += (a.pitch - a.renderArmPitch) * 0.5
		a.renderArmYaw += (a.yaw - a.renderArmYaw) * 0.5
		a.render(snap, target)
		a.window.SwapBuffers()
		if a.fpsWindowStart.IsZero() {
			a.fpsWindowStart = now
		}
		a.fpsFrames++
		elapsedFPS := now.Sub(a.fpsWindowStart)
		if elapsedFPS >= time.Second {
			a.currentFPS = int(math.Round(float64(a.fpsFrames) / elapsedFPS.Seconds()))
			a.fpsFrames = 0
			a.fpsWindowStart = now
		}

		if now.Sub(lastTitleUpdate) >= titleInterval {
			if a.mainMenu {
				a.window.SetTitle("GoMC - Main Menu")
			} else if a.paused {
				a.window.SetTitle("GoMC GUI | PAUSED")
			} else {
				a.window.SetTitle(fmt.Sprintf("GoMC GUI | %d fps | hp=%.1f food=%d pos=(%.1f,%.1f,%.1f) chunks=%d ents=%d",
					a.currentFPS, snap.Health, snap.Food, snap.PlayerX, snap.PlayerY, snap.PlayerZ, snap.LoadedChunks, snap.TrackedEntities))
			}
			lastTitleUpdate = now
		}

		a.syncFrameRate(now)
	}

	return nil
}

func (a *App) advanceAnimatedTextures(now time.Time) {
	for _, tex := range a.blockTextures {
		tex.advanceAnimatedFrame(now)
	}
}

func (a *App) resetRenderTrackFromSnapshot(snap netclient.StateSnapshot) {
	a.renderTrackInit = true
	a.renderPrevX = snap.PlayerX
	a.renderPrevY = snap.PlayerY
	a.renderPrevZ = snap.PlayerZ
	a.renderCurrX = snap.PlayerX
	a.renderCurrY = snap.PlayerY
	a.renderCurrZ = snap.PlayerZ
	a.renderLerpTime = moveTickSeconds
}

func (a *App) syncRenderTrackFromSnapshot(snap netclient.StateSnapshot) bool {
	if !a.renderTrackInit {
		a.resetRenderTrackFromSnapshot(snap)
		return true
	}
	dx := math.Abs(snap.PlayerX-a.renderCurrX) + math.Abs(snap.PlayerY-a.renderCurrY) + math.Abs(snap.PlayerZ-a.renderCurrZ)
	if dx <= 1.0e-9 {
		return false
	}
	// Teleports/corrections should snap immediately to avoid camera smearing.
	if dx > 4.0 {
		a.resetRenderTrackFromSnapshot(snap)
		return true
	}
	a.renderPrevX = a.renderCurrX
	a.renderPrevY = a.renderCurrY
	a.renderPrevZ = a.renderCurrZ
	a.renderCurrX = snap.PlayerX
	a.renderCurrY = snap.PlayerY
	a.renderCurrZ = snap.PlayerZ
	return true
}

func (a *App) interpolatedRenderPlayer(alpha float64, snap netclient.StateSnapshot) (float64, float64, float64) {
	if !a.renderTrackInit {
		return snap.PlayerX, snap.PlayerY, snap.PlayerZ
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	inv := 1.0 - alpha
	x := a.renderPrevX*inv + a.renderCurrX*alpha
	y := a.renderPrevY*inv + a.renderCurrY*alpha
	z := a.renderPrevZ*inv + a.renderCurrZ*alpha
	return x, y, z
}

func (a *App) handleInput(deltaSeconds float64) bool {
	// Keep framebuffer metrics authoritative even if framebuffer callback did not fire yet.
	fbW, fbH := a.window.GetFramebufferSize()
	if fbW <= 0 {
		fbW = 1
	}
	if fbH <= 0 {
		fbH = 1
	}
	if fbW != a.width || fbH != a.height {
		a.width = fbW
		a.height = fbH
		gl.Viewport(0, 0, int32(a.width), int32(a.height))
	}
	a.updateGUIMetrics()
	x, y := a.window.GetCursorPos()
	winW, winH := a.window.GetSize()
	if winW <= 0 {
		winW = 1
	}
	if winH <= 0 {
		winH = 1
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()
	// Translation reference:
	// - net.minecraft.src.GuiScreen.handleMouseInput()
	// Vanilla mapping: mouseX = eventX * scaledWidth / displayWidth
	// (y inversion is not needed here because GLFW cursor Y is top-origin.)
	a.mouseX = int(x * float64(uiW) / float64(winW))
	a.mouseY = int(y * float64(uiH) / float64(winH))
	if a.mouseX < 0 {
		a.mouseX = 0
	} else if a.mouseX >= uiW {
		a.mouseX = uiW - 1
	}
	if a.mouseY < 0 {
		a.mouseY = 0
	} else if a.mouseY >= uiH {
		a.mouseY = uiH - 1
	}

	enterPressed := a.window.GetKey(glfw.KeyEnter) == glfw.Press || a.window.GetKey(glfw.KeyKPEnter) == glfw.Press
	escPressed := a.window.GetKey(glfw.KeyEscape) == glfw.Press
	backspacePressed := a.window.GetKey(glfw.KeyBackspace) == glfw.Press
	inventoryPressed := a.isKeyBindingDown(keyDescInventory)
	chatKeyPressed := a.isKeyBindingDown(keyDescChat)
	commandKeyPressed := a.isKeyBindingDown(keyDescCommand)
	upPressed := a.window.GetKey(glfw.KeyUp) == glfw.Press
	downPressed := a.window.GetKey(glfw.KeyDown) == glfw.Press
	leftMouse := a.window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
	rightMouse := a.window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press
	middleMouse := a.window.GetMouseButton(glfw.MouseButtonMiddle) == glfw.Press
	jumpPressed := a.isKeyBindingDown(keyDescJump)
	sneakPressed := a.isKeyBindingDown(keyDescSneak)
	attackPressed := a.isKeyBindingDown(keyDescAttack)
	usePressed := a.isKeyBindingDown(keyDescUse)
	dropPressed := a.isKeyBindingDown(keyDescDrop)
	pickPressed := a.isKeyBindingDown(keyDescPick)
	playerListPressed := a.isKeyBindingDown(keyDescPlayer)
	sprintPressed := a.window.GetKey(glfw.KeyLeftControl) == glfw.Press || a.window.GetKey(glfw.KeyRightControl) == glfw.Press

	defer func() {
		a.prevLeftMouse = leftMouse
		a.prevRightMouse = rightMouse
		a.prevMiddleMouse = middleMouse
		a.prevEnter = enterPressed
		a.prevSpace = jumpPressed
		a.prevBackspace = backspacePressed
		a.prevE = inventoryPressed
		a.prevT = chatKeyPressed
		a.prevSlash = commandKeyPressed
		a.prevUp = upPressed
		a.prevDown = downPressed
		a.prevAttackInput = attackPressed
		a.prevUseInput = usePressed
		a.prevDropInput = dropPressed
		a.prevPickInput = pickPressed
	}()
	a.showPlayerList = false

	if !a.isCapturingKeyBinding() {
		a.clearKeyPressQueue()
	}
	if a.isCapturingKeyBinding() {
		if escPressed && !a.prevEsc {
			a.setKeyBindingByIndex(a.keyBindCapture, 1)
			a.keyBindCapture = -1
			a.updateKeyBindingButtonsState()
			a.saveOptionsFile()
			a.prevEsc = escPressed
			return true
		}
		if a.tryCaptureKeyBindingFromKeyQueue() {
			return true
		}
	}

	if escPressed && !a.prevEsc && a.mainMenu {
		a.prevEsc = escPressed
		if a.handleMenuEscape() {
			return true
		}
		return false
	}
	if escPressed && !a.prevEsc && !a.mainMenu {
		if a.chatInputOpen {
			a.closeChatInput(true)
			a.prevEsc = escPressed
			return true
		}
		if a.inventoryOpen {
			a.closeInventoryScreen()
			a.prevEsc = escPressed
			return true
		}
		if a.paused {
			if a.pauseScreen == pauseScreenOptions {
				a.pauseScreen = pauseScreenMain
			} else if a.pauseScreen == pauseScreenKeyBindings {
				a.pauseScreen = pauseScreenControls
			} else if a.pauseScreen == pauseScreenVideo || a.pauseScreen == pauseScreenControls || a.pauseScreen == pauseScreenSounds {
				a.pauseScreen = pauseScreenOptions
			} else {
				a.setPaused(false)
			}
		} else {
			a.setPaused(true)
		}
		a.prevEsc = escPressed
		return true
	}
	a.prevEsc = escPressed

	f1Pressed := a.window.GetKey(glfw.KeyF1) == glfw.Press
	if f1Pressed && !a.prevF1 {
		a.hudHidden = !a.hudHidden
	}
	a.prevF1 = f1Pressed

	f3Pressed := a.window.GetKey(glfw.KeyF3) == glfw.Press
	if f3Pressed && !a.prevF3 {
		a.showDebug = !a.showDebug
	}
	a.prevF3 = f3Pressed

	if a.mainMenu {
		a.moveTickAccum = 0
		return a.handleMainMenuInput(leftMouse, rightMouse, middleMouse, enterPressed)
	}
	if a.session == nil {
		a.moveTickAccum = 0
		a.paused = false
		a.pauseScreen = pauseScreenMain
		a.mainMenu = true
		a.menuScreen = menuScreenMain
		a.menuStatus = "No active world session."
		a.applyCursorMode()
		return true
	}

	if a.paused {
		a.moveTickAccum = 0
		if a.pauseScreen == pauseScreenKeyBindings {
			if a.tryCaptureKeyBindingFromMouse(
				leftMouse && !a.prevLeftMouse,
				rightMouse && !a.prevRightMouse,
				middleMouse && !a.prevMiddleMouse,
			) {
				return true
			}
		}
		if leftMouse && !a.prevLeftMouse {
			for _, b := range a.currentPauseButtons() {
				if b == nil || !b.Enabled || !b.contains(a.mouseX, a.mouseY) {
					continue
				}
				if a.soundVolume > 0 {
					audio.PlaySoundKey("random.click", a.soundVolume, 1.0)
				}
				if a.pauseScreen == pauseScreenOptions {
					a.handlePauseOptionButton(b.ID)
				} else if a.pauseScreen == pauseScreenVideo {
					a.handlePauseVideoButton(b.ID)
				} else if a.pauseScreen == pauseScreenControls {
					a.handlePauseControlButton(b.ID)
				} else if a.pauseScreen == pauseScreenKeyBindings {
					a.handlePauseKeybindButton(b.ID)
				} else if a.pauseScreen == pauseScreenSounds {
					a.handlePauseSoundButton(b.ID)
				} else {
					if !a.handlePauseMenuButton(b.ID) {
						return false
					}
				}
				break
			}
		}
		return true
	}

	if a.chatInputOpen {
		a.moveTickAccum = 0
		runes := a.consumeTypedRunes()
		for _, ch := range runes {
			a.appendChatRune(ch)
		}
		if backspacePressed && !a.prevBackspace {
			a.chatInput = trimLastRune(a.chatInput)
		}
		if upPressed && !a.prevUp {
			a.moveChatHistory(-1)
		}
		if downPressed && !a.prevDown {
			a.moveChatHistory(1)
		}
		if enterPressed && !a.prevEnter {
			msg := strings.TrimSpace(a.chatInput)
			if msg != "" {
				a.pushChatHistory(msg)
				_ = a.session.SendChat(msg)
			}
			a.closeChatInput(true)
		}
		return true
	}

	if inventoryPressed && !a.prevE {
		if a.inventoryOpen {
			a.closeInventoryScreen()
		} else {
			a.openInventoryScreen()
		}
		return true
	}
	if a.inventoryOpen {
		a.moveTickAccum = 0
		if leftMouse && !a.prevLeftMouse {
			a.handleInventoryClick(false, sneakPressed)
		}
		if rightMouse && !a.prevRightMouse {
			a.handleInventoryClick(true, sneakPressed)
		}
		return true
	}
	a.showPlayerList = playerListPressed

	if (chatKeyPressed && !a.prevT) || (enterPressed && !a.prevEnter) {
		a.openChatInput("")
		return true
	}
	if commandKeyPressed && !a.prevSlash {
		a.openChatInput("/")
		return true
	}
	if dropPressed && !a.prevDropInput {
		_ = a.session.DropHeldItem(sprintPressed)
	}
	if pickPressed && !a.prevPickInput {
		a.handlePickBlockAction()
	}

	snapMove := a.session.Snapshot()
	allowFlight := snapMove.CanFly || snapMove.IsCreative
	flyingNow := snapMove.IsFlying
	if allowFlight && jumpPressed && !a.prevSpace {
		if !a.lastSpaceTap.IsZero() && time.Since(a.lastSpaceTap) <= 250*time.Millisecond {
			flyingNow = !flyingNow
			_ = a.session.SetFlying(flyingNow)
			a.lastSpaceTap = time.Time{}
		} else {
			a.lastSpaceTap = time.Now()
		}
	}

	if a.firstMouse {
		a.lastMouseX = x
		a.lastMouseY = y
		a.firstMouse = false
	}

	dxMouse := x - a.lastMouseX
	dyMouse := y - a.lastMouseY
	a.lastMouseX = x
	a.lastMouseY = y

	if dxMouse != 0 || dyMouse != 0 {
		turnScale := a.mouseTurnScale()
		a.yaw += dxMouse * turnScale
		pitchSign := 1.0
		if a.invertMouse {
			pitchSign = -1.0
		}
		// Match vanilla: invert option flips pitch sign only.
		a.pitch += dyMouse * turnScale * pitchSign
		if a.pitch > 89.9 {
			a.pitch = 89.9
		}
		if a.pitch < -89.9 {
			a.pitch = -89.9
		}
		_ = a.session.Look(float32(a.yaw), float32(a.pitch))
	}

	if attackPressed && !a.prevAttackInput {
		a.startHandSwing()
		_ = a.session.SwingArm()
		snapNow := a.session.Snapshot()
		blockHit := a.pickBlockTarget(snapNow, interactReach)
		entityHit := a.pickEntityTarget(snapNow, interactReach)
		blockDist := a.blockTargetDistance(snapNow, blockHit)

		if entityHit.Hit && (!blockHit.Hit || entityHit.Dist <= blockDist) {
			_ = a.session.UseEntity(entityHit.EntityID, true)
		} else if blockHit.Hit {
			id, _, ok := a.session.BlockAt(blockHit.X, blockHit.Y, blockHit.Z)
			_ = a.session.DigBlock(int32(blockHit.X), int32(blockHit.Y), int32(blockHit.Z), blockHit.Face)
			if ok && id > 0 && !block.IsLiquid(id) {
				audio.PlayDigBlock(id)
			}
		}
	}

	if usePressed && !a.prevUseInput {
		snapNow := a.session.Snapshot()
		target := a.pickBlockTarget(snapNow, interactReach)
		if target.Hit {
			_ = a.session.PlaceHeldBlock(int32(target.X), int32(target.Y), int32(target.Z), target.Face)
			audio.PlayPlaceHeldItem(int(snapNow.HeldItemID))
		}
	}

	for i := 0; i < 9; i++ {
		key := glfw.Key(int(glfw.Key1) + i)
		pressed := a.window.GetKey(key) == glfw.Press
		if pressed && !a.prevDigit[i] {
			_ = a.session.SelectHotbar(int16(i))
		}
		a.prevDigit[i] = pressed
	}

	if sneakPressed != a.prevSneakKey {
		_ = a.session.SetSneaking(sneakPressed)
		a.prevSneakKey = sneakPressed
	}

	forward := 0.0
	strafe := 0.0
	if a.isKeyBindingDown(keyDescForward) {
		forward += 1.0
	}
	if a.isKeyBindingDown(keyDescBack) {
		forward -= 1.0
	}
	if a.isKeyBindingDown(keyDescLeft) {
		strafe += 1.0
	}
	if a.isKeyBindingDown(keyDescRight) {
		strafe -= 1.0
	}
	a.moveTickAccum += deltaSeconds
	maxAccum := moveTickSeconds * float64(maxMoveCatchUp)
	if a.moveTickAccum > maxAccum {
		a.moveTickAccum = maxAccum
	}
	for a.moveTickAccum >= moveTickSeconds {
		a.applyMovementTick(
			moveTickSeconds,
			forward,
			strafe,
			jumpPressed,
			sneakPressed,
			sprintPressed,
			allowFlight,
			flyingNow,
		)
		a.moveTickAccum -= moveTickSeconds
	}
	return true
}

func (a *App) applyMovementTick(
	stepSeconds float64,
	forward, strafe float64,
	jumpPressed, sneakPressed, sprintPressed bool,
	allowFlight, flyingNow bool,
) {
	if a.session == nil {
		return
	}

	snapMove := a.session.Snapshot()
	grounded := a.playerGroundedAt(snapMove.PlayerX, snapMove.PlayerY, snapMove.PlayerZ)
	a.tickSprintState(forward, sneakPressed, sprintPressed, grounded, allowFlight, snapMove.Food)

	speed := a.moveSpeed * stepSeconds
	if a.localSprinting {
		speed *= 1.30
	}
	if sneakPressed {
		speed *= 0.30
	}

	dxDesired, dzDesired := movementDeltaFromYaw(a.yaw, forward, strafe, speed)
	dyDesired := 0.0

	if allowFlight && flyingNow {
		a.velY = 0
		if jumpPressed {
			dyDesired += speed
		}
		if sneakPressed {
			dyDesired -= speed
		}
	} else {
		ticks := stepSeconds * 20.0
		if grounded {
			// Translation reference:
			// - net.minecraft.src.MovementInputFromOptions.jump (hold-to-jump behavior)
			// - net.minecraft.src.EntityLivingBase.moveEntityWithHeading()
			if jumpPressed {
				a.velY = jumpVelocity
			} else if a.velY < 0 {
				a.velY = 0
			}
		} else {
			a.velY = (a.velY - gravityPerTick*ticks) * math.Pow(dragPerTick, ticks)
		}
		dyDesired = a.velY * ticks
	}

	dxMove, dyMove, dzMove, groundedAfter := a.resolvePlayerMovement(
		snapMove.PlayerX,
		snapMove.PlayerY,
		snapMove.PlayerZ,
		dxDesired,
		dyDesired,
		dzDesired,
	)
	if dyDesired < 0 && dyMove > dyDesired+1e-6 {
		a.velY = 0
	}
	a.localOnGround = groundedAfter
	a.session.SetLocalOnGround(groundedAfter)
	collidedHoriz := (math.Abs(dxMove-dxDesired) > 1.0e-6 || math.Abs(dzMove-dzDesired) > 1.0e-6) &&
		(math.Abs(dxDesired) > 1.0e-9 || math.Abs(dzDesired) > 1.0e-9)
	if a.localSprinting && collidedHoriz {
		a.setLocalSprinting(false)
	}
	a.updateViewBobbingState(dxMove, dyMove, dzMove, groundedAfter, snapMove.Health <= 0)
	if dxMove != 0 || dyMove != 0 || dzMove != 0 {
		_ = a.session.MoveRelative(dxMove, dyMove, dzMove)
	}
}

// Translation reference:
// - net.minecraft.src.EntityPlayerSP.onLivingUpdate()
// - net.minecraft.src.EntityRenderer.setupViewBobbing(float)
func (a *App) updateViewBobbingState(dxMove, dyMove, dzMove float64, onGround, dead bool) {
	a.prevWalkDistance = a.walkDistance
	a.prevCameraYaw = a.cameraYaw
	a.prevCameraPitch = a.cameraPitch

	step := math.Sqrt(dxMove*dxMove+dzMove*dzMove) * 0.6
	if step > 1.0 {
		step = 1.0
	}
	a.walkDistance += step

	targetYaw := math.Sqrt(dxMove*dxMove + dzMove*dzMove)
	if targetYaw > 0.1 {
		targetYaw = 0.1
	}
	targetPitch := math.Atan(-dyMove*0.2) * 15.0
	if !onGround || dead {
		targetYaw = 0.0
	}
	if onGround || dead {
		targetPitch = 0.0
	}
	a.cameraYaw += (targetYaw - a.cameraYaw) * 0.4
	a.cameraPitch += (targetPitch - a.cameraPitch) * 0.8
}

// Translation reference:
// - net.minecraft.src.EntityPlayerSP.onLivingUpdate()
func (a *App) tickSprintState(
	forward float64,
	sneakPressed, sprintPressed, grounded, allowFlight bool,
	foodLevel int16,
) {
	if a.sprintTimer > 0 {
		a.sprintTimer--
	}

	forwardPressed := forward > 0.0
	justPressedForward := forwardPressed && !a.prevForwardKey
	canSprintFood := foodLevel > 6 || allowFlight
	canSprintNow := forwardPressed && !sneakPressed && canSprintFood

	if a.localSprinting {
		if !canSprintNow {
			a.setLocalSprinting(false)
		}
	} else if canSprintNow && grounded {
		if sprintPressed {
			a.setLocalSprinting(true)
			a.sprintTimer = 0
		} else if justPressedForward {
			if a.sprintTimer > 0 {
				a.setLocalSprinting(true)
				a.sprintTimer = 0
			} else {
				a.sprintTimer = sprintTapTicks
			}
		}
	}

	if !forwardPressed {
		a.sprintTimer = 0
	}
	a.prevForwardKey = forwardPressed
}

func (a *App) setLocalSprinting(enabled bool) {
	if a.localSprinting == enabled {
		return
	}
	a.localSprinting = enabled
	if a.session != nil {
		_ = a.session.SetSprinting(enabled)
	}
}

func (a *App) setPaused(paused bool) {
	a.paused = paused
	a.moveTickAccum = 0
	a.firstMouse = true
	a.keyBindCapture = -1
	a.clearKeyPressQueue()
	if paused {
		a.pauseScreen = pauseScreenMain
		a.initPauseButtons()
		a.initPauseOptionsButtons()
	} else {
		a.pauseScreen = pauseScreenMain
	}
	a.applyCursorMode()
}

func (a *App) currentPauseButtons() []*guiButton {
	if a.pauseScreen == pauseScreenOptions {
		return a.pauseOptionButtons
	}
	if a.pauseScreen == pauseScreenVideo {
		return a.videoButtons
	}
	if a.pauseScreen == pauseScreenControls {
		return a.controlButtons
	}
	if a.pauseScreen == pauseScreenKeyBindings {
		return a.keyBindButtons
	}
	if a.pauseScreen == pauseScreenSounds {
		return a.soundButtons
	}
	return a.pauseButtons
}

func (a *App) handlePauseMenuButton(id int) bool {
	switch id {
	case buttonIDPauseReturnToGame:
		a.setPaused(false)
	case buttonIDPauseDisconnect:
		a.disconnectToMainMenu("")
	case buttonIDPauseOptions:
		a.pauseScreen = pauseScreenOptions
		a.updatePauseOptionButtonsState()
	case buttonIDPauseAchievements:
		a.menuStatus = "Achievements screen is not implemented yet."
	case buttonIDPauseStats:
		a.menuStatus = "Statistics screen is not implemented yet."
	case buttonIDPauseShareToLAN:
		a.menuStatus = "Open to LAN is not implemented yet."
	}
	return true
}

func (a *App) handlePauseOptionButton(id int) {
	changed := false
	switch id {
	case buttonIDOptionDone:
		a.pauseScreen = pauseScreenMain
		a.menuStatus = ""
	case buttonIDOptionDifficulty:
		a.optionDifficulty = (a.optionDifficulty + 1) & 3
		changed = true
	case buttonIDOptionVideo:
		a.pauseScreen = pauseScreenVideo
		a.menuStatus = ""
	case buttonIDOptionControls:
		a.pauseScreen = pauseScreenControls
		a.keyBindCapture = -1
		a.menuStatus = ""
	case buttonIDOptionLanguage:
		a.menuStatus = "Language screen is not implemented yet."
	case buttonIDOptionMusic:
		a.pauseScreen = pauseScreenSounds
		a.keyBindCapture = -1
		a.menuStatus = ""
	case buttonIDOptionSnooper:
		a.menuStatus = "Snooper Settings are not implemented yet."
	}
	a.updatePauseOptionButtonsState()
	a.updateVideoButtonsState()
	a.updateControlButtonsState()
	a.updateSoundButtonsState()
	if changed {
		a.saveOptionsFile()
	}
}

func (a *App) handlePauseVideoButton(id int) {
	changed := false
	switch id {
	case buttonIDVideoDone:
		a.pauseScreen = pauseScreenOptions
		a.menuStatus = ""
	case buttonIDVideoGraphics:
		a.fancyGraphics = !a.fancyGraphics
		changed = true
	case buttonIDVideoClouds:
		a.cloudsEnabled = !a.cloudsEnabled
		changed = true
	case buttonIDVideoRDMinus:
		if a.renderDistance < 3 {
			a.renderDistance++
			changed = true
		}
	case buttonIDVideoRDPlus:
		if a.renderDistance > 0 {
			a.renderDistance--
			changed = true
		}
	case buttonIDVideoFOVMinus:
		a.fovSetting -= 0.05
		if a.fovSetting < 0.0 {
			a.fovSetting = 0.0
		}
		changed = true
	case buttonIDVideoFOVPlus:
		a.fovSetting += 0.05
		if a.fovSetting > 1.0 {
			a.fovSetting = 1.0
		}
		changed = true
	case buttonIDVideoFPS:
		a.limitFramerateMode = (a.limitFramerateMode + 1) % len(framerateModeNames)
		changed = true
	case buttonIDVideoGUIScale:
		a.guiScaleMode = (a.guiScaleMode + 1) % len(guiScaleModeNames)
		a.updateGUIMetrics()
		changed = true
	case buttonIDVideoViewBobbing:
		a.viewBobbing = !a.viewBobbing
		changed = true
	}
	a.updatePauseOptionButtonsState()
	a.updateVideoButtonsState()
	a.updateControlButtonsState()
	a.updateSoundButtonsState()
	if changed {
		a.saveOptionsFile()
	}
}

func (a *App) handlePauseControlButton(id int) {
	changed := false
	switch id {
	case buttonIDControlDone:
		a.pauseScreen = pauseScreenOptions
		a.keyBindCapture = -1
		a.menuStatus = ""
	case buttonIDControlSensMinus:
		a.mouseSens -= 0.02
		if a.mouseSens < 0.0 {
			a.mouseSens = 0.0
		}
		changed = true
	case buttonIDControlSensPlus:
		a.mouseSens += 0.02
		if a.mouseSens > 1.0 {
			a.mouseSens = 1.0
		}
		changed = true
	case buttonIDControlInvert:
		a.invertMouse = !a.invertMouse
		changed = true
	case buttonIDControlKeybinds:
		a.pauseScreen = pauseScreenKeyBindings
		a.keyBindCapture = -1
		a.clearKeyPressQueue()
		a.menuStatus = ""
	case buttonIDControlTouchscreen:
		a.menuStatus = "Touchscreen mode is not available on desktop."
	}
	a.updateControlButtonsState()
	a.updateKeyBindingButtonsState()
	a.updateSoundButtonsState()
	if changed {
		a.saveOptionsFile()
	}
}

func (a *App) handlePauseKeybindButton(id int) {
	if id == buttonIDKeybindDone {
		a.pauseScreen = pauseScreenControls
		a.keyBindCapture = -1
		a.clearKeyPressQueue()
		a.menuStatus = ""
		return
	}
	if id >= buttonIDKeybindBase {
		idx := id - buttonIDKeybindBase
		if idx >= 0 && idx < len(a.keyBindings) {
			a.keyBindCapture = idx
			a.clearKeyPressQueue()
			a.updateKeyBindingButtonsState()
		}
	}
}

func (a *App) handlePauseSoundButton(id int) {
	changed := false
	switch id {
	case buttonIDSoundDone:
		a.pauseScreen = pauseScreenOptions
		a.menuStatus = ""
	case buttonIDSoundMusicMinus:
		if a.musicVolume > 0.0 {
			a.musicVolume = clampFloat64(a.musicVolume-0.1, 0.0, 1.0)
			changed = true
		}
	case buttonIDSoundMusicPlus:
		if a.musicVolume < 1.0 {
			a.musicVolume = clampFloat64(a.musicVolume+0.1, 0.0, 1.0)
			changed = true
		}
	case buttonIDSoundSoundMinus:
		if a.soundVolume > 0.0 {
			a.soundVolume = clampFloat64(a.soundVolume-0.1, 0.0, 1.0)
			changed = true
		}
	case buttonIDSoundSoundPlus:
		if a.soundVolume < 1.0 {
			a.soundVolume = clampFloat64(a.soundVolume+0.1, 0.0, 1.0)
			changed = true
		}
	}
	a.updateSoundButtonsState()
	if changed {
		a.saveOptionsFile()
	}
}

func (a *App) disconnectToMainMenu(status string) {
	if a.session != nil {
		a.replaceSession(nil)
	}
	a.mainMenu = true
	a.paused = false
	a.pauseScreen = pauseScreenMain
	a.menuScreen = menuScreenMain
	a.inventoryOpen = false
	a.showPlayerList = false
	a.chatInputOpen = false
	a.chatInput = ""
	a.chatDraft = ""
	a.keyBindCapture = -1
	a.clearKeyPressQueue()
	if status != "" {
		a.menuStatus = status
	} else {
		a.menuStatus = ""
	}
	a.firstMouse = true
	a.initAllMenuButtons()
	a.applyCursorMode()
}

func (a *App) openChatInput(initial string) {
	if a.session == nil || a.mainMenu || a.paused {
		return
	}
	a.chatInputOpen = true
	a.chatInput = initial
	a.chatDraft = initial
	a.chatHistPos = len(a.chatHistory)
	a.inventoryOpen = false
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.firstMouse = true
	a.moveTickAccum = 0
	a.applyCursorMode()
}

func (a *App) closeChatInput(clear bool) {
	if !a.chatInputOpen {
		return
	}
	a.chatInputOpen = false
	if clear {
		a.chatInput = ""
	}
	a.chatDraft = ""
	a.chatHistPos = len(a.chatHistory)
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.firstMouse = true
	a.applyCursorMode()
}

func (a *App) appendChatRune(ch rune) {
	if ch < 32 || ch == 127 {
		return
	}
	if ch == '\n' || ch == '\r' {
		return
	}
	if len([]rune(a.chatInput)) >= 100 {
		return
	}
	a.chatInput += string(ch)
	if a.chatHistPos >= len(a.chatHistory) {
		a.chatDraft = a.chatInput
	}
}

func (a *App) pushChatHistory(message string) {
	if message == "" {
		return
	}
	a.chatHistory = append(a.chatHistory, message)
	if len(a.chatHistory) > 100 {
		a.chatHistory = a.chatHistory[len(a.chatHistory)-100:]
	}
	a.chatHistPos = len(a.chatHistory)
	a.chatDraft = ""
}

func (a *App) moveChatHistory(delta int) {
	if len(a.chatHistory) == 0 || delta == 0 {
		return
	}
	if a.chatHistPos < 0 || a.chatHistPos > len(a.chatHistory) {
		a.chatHistPos = len(a.chatHistory)
	}
	if delta < 0 {
		if a.chatHistPos == len(a.chatHistory) {
			a.chatDraft = a.chatInput
		}
		if a.chatHistPos > 0 {
			a.chatHistPos--
		}
		a.chatInput = a.chatHistory[a.chatHistPos]
		return
	}
	if a.chatHistPos < len(a.chatHistory)-1 {
		a.chatHistPos++
		a.chatInput = a.chatHistory[a.chatHistPos]
		return
	}
	if a.chatHistPos == len(a.chatHistory)-1 {
		a.chatHistPos = len(a.chatHistory)
		a.chatInput = a.chatDraft
	}
}

func (a *App) openInventoryScreen() {
	if a.session == nil || a.mainMenu || a.paused {
		return
	}
	a.inventoryOpen = true
	a.chatInputOpen = false
	a.chatInput = ""
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.firstMouse = true
	a.moveTickAccum = 0
	a.applyCursorMode()
}

func (a *App) closeInventoryScreen() {
	if !a.inventoryOpen {
		return
	}
	a.inventoryOpen = false
	a.firstMouse = true
	a.applyCursorMode()
	if a.session != nil {
		_ = a.session.CloseInventoryWindow()
	}
}

func (a *App) handleInventoryClick(rightClick bool, shift bool) {
	if a.session == nil {
		return
	}
	slot := a.playerInventorySlotAt(a.mouseX, a.mouseY)
	_ = a.session.ClickWindowSlot(slot, rightClick, shift)
}

func (a *App) playerInventorySlotAt(mouseX, mouseY int) int16 {
	slot := int16(-999)
	a.forEachPlayerInventorySlot(func(candidate int16, x, y int) {
		if slot != -999 {
			return
		}
		if mouseX >= x && mouseX < x+16 && mouseY >= y && mouseY < y+16 {
			slot = candidate
		}
	})
	return slot
}

func (a *App) playerInventoryScreenOrigin() (int, int) {
	return a.uiWidth()/2 - 88, a.uiHeight()/2 - 83
}

func (a *App) forEachPlayerInventorySlot(fn func(slot int16, x, y int)) {
	left, top := a.playerInventoryScreenOrigin()
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			fn(int16(9+row*9+col), left+8+col*18, top+84+row*18)
		}
	}
	for col := 0; col < 9; col++ {
		fn(int16(36+col), left+8+col*18, top+142)
	}
}

func (a *App) drawInventorySlotText(stack netclient.InventorySlotSnapshot, x, y int) {
	if a.font == nil || stack.ItemID <= 0 || stack.StackSize <= 0 {
		return
	}
	if stack.StackSize > 1 {
		count := strconv.Itoa(int(stack.StackSize))
		a.font.drawStringWithShadow(count, x+17-a.font.getStringWidth(count), y+9, 0xFFFFFF)
	}
}

func (a *App) inventoryHoverLabel(stack netclient.InventorySlotSnapshot) string {
	if stack.ItemID <= 0 || stack.StackSize <= 0 {
		return ""
	}
	if stack.StackSize > 1 {
		return fmt.Sprintf("ID %d x%d", stack.ItemID, stack.StackSize)
	}
	return fmt.Sprintf("ID %d", stack.ItemID)
}

func (a *App) drawInventoryScreen() {
	if a.session == nil {
		return
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()
	left, top := a.playerInventoryScreenOrigin()

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(uiW, uiH)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	drawGradientRect(0, 0, uiW, uiH, 0x80101010, 0x80101010)
	if a.texInventory != nil {
		gl.Enable(gl.TEXTURE_2D)
		gl.Color4f(1, 1, 1, 1)
		drawTexturedRect(a.texInventory, float32(left), float32(top), 176, 166, 0, 0, 176, 166)
	} else {
		drawSolidRect(left, top, left+176, top+166, 0xC0101010)
	}
	a.drawInventoryPlayerPreview(left, top)

	inv := a.session.InventorySnapshot()
	hoveredSlot := int16(-1)
	a.forEachPlayerInventorySlot(func(slot int16, x, y int) {
		if a.mouseX >= x && a.mouseX < x+16 && a.mouseY >= y && a.mouseY < y+16 {
			hoveredSlot = slot
			drawSolidRect(x, y, x+16, y+16, 0x80FFFFFF)
		}
		if int(slot) >= 0 && int(slot) < len(inv) {
			a.drawInventorySlotText(inv[int(slot)], x, y)
		}
	})

	if hoveredSlot >= 0 && int(hoveredSlot) < len(inv) && a.font != nil {
		label := a.inventoryHoverLabel(inv[int(hoveredSlot)])
		if label != "" {
			w := a.font.getStringWidth(label)
			tx := a.mouseX + 10
			ty := a.mouseY + 8
			if tx+w+6 > uiW {
				tx = uiW - w - 6
			}
			if ty+12 > uiH {
				ty = uiH - 12
			}
			drawSolidRect(tx-3, ty-3, tx+w+3, ty+9, 0xF0100010)
			a.font.drawStringWithShadow(label, tx, ty, 0xFFFFFF)
		}
	}

	if a.font != nil {
		mode := "Survival"
		if a.session.Snapshot().IsCreative {
			mode = "Creative"
		}
		a.font.drawStringWithShadow("Inventory", left+8, top+6, 0xFFFFFF)
		a.font.drawStringWithShadow("Mode: "+mode+"  use /gamemode 0|1", left+8, top+72, 0xA0A0A0)
	}

	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

// Translation reference:
// - net.minecraft.src.GuiInventory.drawScreen()
// - net.minecraft.src.GuiInventory.func_110423_a(...) (drawEntityOnScreen)
func (a *App) drawInventoryPlayerPreview(left, top int) {
	uiW, uiH := a.uiWidth(), a.uiHeight()
	previewX := float32(left + 51)
	previewY := float32(top + 75)
	scale := float32(30.0)

	dx := previewX - float32(a.mouseX)
	dy := (previewY - 50.0) - float32(a.mouseY)
	bodyYaw := float32(math.Atan(float64(dx/40.0)) * 20.0)
	headYaw := float32(math.Atan(float64(dx/40.0)) * 40.0)
	headRelYaw := normalizeDegrees(headYaw - bodyYaw)
	pitch := -float32(math.Atan(float64(dy/40.0)) * 20.0)

	profile := modelProfileForEntityType(0)
	tex := a.entityTextureForType(0)

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0, float64(uiW), float64(uiH), 0, -200, 200)

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	gl.LoadIdentity()

	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.ALPHA_TEST)
	gl.AlphaFunc(gl.GREATER, 0.1)
	gl.Enable(gl.TEXTURE_2D)

	gl.Translatef(previewX, previewY, 100.0)
	gl.Scalef(-scale, scale, scale)
	gl.Rotatef(180.0, 0, 0, 1)
	gl.Rotatef(135.0, 0, 1, 0)
	gl.Rotatef(-135.0, 0, 1, 0)
	gl.Rotatef(pitch, 1, 0, 0)
	gl.Rotatef(180.0-bodyYaw, 0, 1, 0)
	drawBipedModel(profile, tex, headRelYaw, pitch, 0, false)

	gl.Color4f(1, 1, 1, 1)

	gl.PopMatrix()
	gl.MatrixMode(gl.PROJECTION)
	gl.PopMatrix()
	gl.MatrixMode(gl.MODELVIEW)

	// Restore 2D GUI render state.
	begin2D(uiW, uiH)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}

func (a *App) initPauseButtons() {
	w := a.uiWidth()
	h := a.uiHeight()
	offset := -16
	baseY := h / 4
	quitLabel := "Disconnect"
	if a.activeWorld != "" {
		quitLabel = "Save and Quit to Title"
	}
	a.pauseButtons = []*guiButton{
		newButton(buttonIDPauseDisconnect, w/2-100, baseY+120+offset, 200, 20, quitLabel),
		newButton(buttonIDPauseReturnToGame, w/2-100, baseY+24+offset, 200, 20, "Return to Game"),
		newButton(buttonIDPauseAchievements, w/2-100, baseY+48+offset, 98, 20, "Achievements"),
		newButton(buttonIDPauseStats, w/2+2, baseY+48+offset, 98, 20, "Statistics"),
		newButton(buttonIDPauseOptions, w/2-100, baseY+96+offset, 98, 20, "Options..."),
		newButton(buttonIDPauseShareToLAN, w/2+2, baseY+96+offset, 98, 20, "Open to LAN"),
	}
	// Not implemented yet.
	a.pauseButtons[2].Enabled = false
	a.pauseButtons[3].Enabled = false
	a.pauseButtons[5].Enabled = false
}

func (a *App) initPauseOptionsButtons() {
	a.initOptionButtons()
	a.initVideoButtons()
	a.initControlButtons()
	a.initKeyBindingButtons()
	a.initSoundButtons()
	a.updatePauseOptionButtonsState()
	a.updateVideoButtonsState()
	a.updateControlButtonsState()
	a.updateKeyBindingButtonsState()
	a.updateSoundButtonsState()
	a.pauseOptionButtons = append(a.pauseOptionButtons[:0], a.optionButtons...)
}

func (a *App) updatePauseOptionButtonsState() {
	wasMainMenu := a.mainMenu
	a.mainMenu = true
	a.updateOptionButtonsState()
	a.mainMenu = wasMainMenu
	a.pauseOptionButtons = append(a.pauseOptionButtons[:0], a.optionButtons...)
}

func (a *App) initMainButtons() {
	w := a.uiWidth()
	h := a.uiHeight()
	baseY := h/4 + 48
	a.mainButtons = []*guiButton{
		newButton(buttonIDMenuSingleplayer, w/2-100, baseY, 200, 20, "Singleplayer"),
		newButton(buttonIDMenuMultiplayer, w/2-100, baseY+24, 200, 20, "Multiplayer"),
		newButton(buttonIDMenuOnline, w/2-100, baseY+48, 200, 20, "Minecraft Realms"),
		newButton(buttonIDMenuOptions, w/2-100, baseY+72+12, 98, 20, "Options..."),
		newButton(buttonIDMenuQuit, w/2+2, baseY+72+12, 98, 20, "Quit Game"),
		newButton(buttonIDMenuLanguage, w/2-124, baseY+72+12, 20, 20, "L"),
	}
	// Realms and language pages are not implemented yet.
	a.mainButtons[2].Enabled = false
	a.mainButtons[5].Enabled = false
}

func (a *App) applyCursorMode() {
	if a.window == nil {
		return
	}
	if a.paused || a.mainMenu || a.inventoryOpen || a.chatInputOpen {
		a.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		return
	}
	a.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
}

func (a *App) replaceSession(next *netclient.Session) {
	old := a.session
	a.releaseChunkRenderCache()
	a.session = next
	if next != nil {
		a.eventsCh = next.Events()
		a.chatClosed = false
		snap := next.Snapshot()
		a.yaw = float64(snap.PlayerYaw)
		a.pitch = float64(snap.PlayerPitch)
	} else {
		a.eventsCh = nil
		a.chatClosed = true
	}
	a.firstMouse = true
	a.moveTickAccum = 0
	a.lastOnGround = time.Time{}
	a.localOnGround = false
	a.velY = 0
	a.prevLeftMouse = false
	a.prevRightMouse = false
	a.prevMiddleMouse = false
	a.prevEnter = false
	a.prevSpace = false
	a.prevBackspace = false
	a.prevE = false
	a.prevT = false
	a.prevSlash = false
	a.prevUp = false
	a.prevDown = false
	a.prevSneakKey = false
	a.prevAttackInput = false
	a.prevUseInput = false
	a.prevDropInput = false
	a.prevPickInput = false
	a.prevForwardKey = false
	a.sprintTimer = 0
	a.localSprinting = false
	a.renderTrackInit = false
	a.walkDistance = 0
	a.prevWalkDistance = 0
	a.cameraYaw = 0
	a.prevCameraYaw = 0
	a.cameraPitch = 0
	a.prevCameraPitch = 0
	for i := range a.prevDigit {
		a.prevDigit[i] = false
	}
	if next != nil {
		a.resetRenderTrackFromSnapshot(next.Snapshot())
	}
	a.handSwingStart = time.Time{}
	a.paused = false
	a.pauseScreen = pauseScreenMain
	a.inventoryOpen = false
	a.chatInputOpen = false
	a.showPlayerList = false
	a.chatInput = ""
	a.chatDraft = ""
	a.chatHistPos = len(a.chatHistory)
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.keyPressQueue = a.keyPressQueue[:0]
	a.keyBindCapture = -1
	a.applyCursorMode()

	if old != nil && old != next {
		_ = old.Close()
		_ = old.Wait()
	}
}

func (a *App) render(snap netclient.StateSnapshot, target blockTarget) {
	if a.mainMenu {
		gl.ClearColor(0.10, 0.12, 0.16, 1.0)
		a.drawMenuScreen()
		return
	}

	alpha := a.renderLerpTime / moveTickSeconds
	if alpha < 0 {
		alpha = 0
	} else if alpha > 1 {
		alpha = 1
	}
	partial := float32(alpha)
	skyR, skyG, skyB := worldSkyColor(snap.WorldTime, partial, 0.8)
	gl.ClearColor(skyR, skyG, skyB, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.ALPHA_TEST)
	gl.AlphaFunc(gl.GREATER, 0.1)

	aspect := float64(a.width) / float64(maxInt(a.height, 1))
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	setPerspective(a.currentFOVDegrees(), aspect, 0.05, 256.0)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	a.applyViewBobbingTransform(partial)
	gl.Rotatef(float32(a.pitch), 1, 0, 0)
	gl.Rotatef(float32(a.yaw+180.0), 0, 1, 0)
	a.drawSky(snap, partial)
	if a.shouldRenderClouds() {
		a.drawCloudLayer(snap, partial)
	}
	camX, camY, camZ := a.interpolatedRenderPlayer(alpha, snap)
	gl.Translatef(float32(-camX), float32(-(camY + playerEyeHeight)), float32(-camZ))
	a.drawWorld(snap)
	a.drawEntities()
	if target.Hit {
		drawBlockOutlineAABB(
			float32(target.MinX),
			float32(target.MinY),
			float32(target.MinZ),
			float32(target.MaxX),
			float32(target.MaxY),
			float32(target.MaxZ),
		)
	}
	// Vanilla EntityRenderer clears depth before first-person hand render.
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.drawFirstPersonArm(snap)
	a.drawHUD(snap)
	if a.inventoryOpen {
		a.drawInventoryScreen()
	}
	if a.paused {
		a.drawPauseMenu()
	}
}

// Translation reference:
// - net.minecraft.src.EntityRenderer.setupViewBobbing(float)
func (a *App) applyViewBobbingTransform(partial float32) {
	if !a.viewBobbing {
		return
	}
	var3 := float32(a.walkDistance - a.prevWalkDistance)
	var4 := float32(-(a.walkDistance + float64(var3)*float64(partial)))
	var5 := float32(a.prevCameraYaw + (a.cameraYaw-a.prevCameraYaw)*float64(partial))
	var6 := float32(a.prevCameraPitch + (a.cameraPitch-a.prevCameraPitch)*float64(partial))
	walkPi := float64(var4) * math.Pi
	sinWalk := float32(math.Sin(walkPi))
	cosWalk := float32(math.Cos(walkPi))

	gl.Translatef(
		sinWalk*var5*0.5,
		-float32(math.Abs(float64(cosWalk*var5))),
		0.0,
	)
	gl.Rotatef(sinWalk*var5*3.0, 0.0, 0.0, 1.0)
	gl.Rotatef(float32(math.Abs(math.Cos(walkPi-0.2)*float64(var5)))*5.0, 1.0, 0.0, 0.0)
	gl.Rotatef(var6, 1.0, 0.0, 0.0)
}

// Translation reference:
// - net.minecraft.src.RenderGlobal.renderSky(...)
// - net.minecraft.src.WorldProvider.calculateCelestialAngle(...)
func (a *App) drawSky(snap netclient.StateSnapshot, partial float32) {
	celestial := celestialAngle(snap.WorldTime, partial)
	sr, sg, sb, sa, hasSunrise := sunriseSunsetColors(celestial)
	skyR, skyG, skyB := worldSkyColor(snap.WorldTime, partial, 0.8)
	star := starBrightness(snap.WorldTime, partial)

	gl.PushMatrix()
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.ALPHA_TEST)
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
	gl.Disable(gl.TEXTURE_2D)
	gl.Color3f(skyR, skyG, skyB)
	if a.skyListID != 0 {
		gl.CallList(a.skyListID)
	} else {
		a.drawSkyGradientDome(snap.WorldTime, partial)
	}

	if hasSunrise {
		gl.Disable(gl.TEXTURE_2D)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.ShadeModel(gl.SMOOTH)
		gl.PushMatrix()
		gl.Rotatef(90.0, 1.0, 0.0, 0.0)
		if math.Sin(float64(celestial)*math.Pi*2.0) < 0 {
			gl.Rotatef(180.0, 0.0, 0.0, 1.0)
		}
		gl.Rotatef(90.0, 0.0, 0.0, 1.0)
		gl.Begin(gl.TRIANGLE_FAN)
		gl.Color4f(sr, sg, sb, sa)
		gl.Vertex3f(0.0, 100.0, 0.0)
		const slices = 16
		gl.Color4f(sr, sg, sb, 0.0)
		for i := 0; i <= slices; i++ {
			ang := float64(i) * math.Pi * 2.0 / float64(slices)
			sinA := float32(math.Sin(ang))
			cosA := float32(math.Cos(ang))
			gl.Vertex3f(sinA*120.0, cosA*120.0, -cosA*40.0*sa)
		}
		gl.End()
		gl.PopMatrix()
		gl.ShadeModel(gl.FLAT)
	}

	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	gl.PushMatrix()
	gl.Rotatef(-90.0, 0.0, 1.0, 0.0)
	gl.Rotatef(celestial*360.0, 1.0, 0.0, 0.0)
	gl.Color4f(1.0, 1.0, 1.0, 1.0)

	if a.texSun != nil {
		const sunSize = float32(30.0)
		a.texSun.bind()
		gl.Begin(gl.QUADS)
		gl.TexCoord2f(0.0, 0.0)
		gl.Vertex3f(-sunSize, 100.0, -sunSize)
		gl.TexCoord2f(1.0, 0.0)
		gl.Vertex3f(sunSize, 100.0, -sunSize)
		gl.TexCoord2f(1.0, 1.0)
		gl.Vertex3f(sunSize, 100.0, sunSize)
		gl.TexCoord2f(0.0, 1.0)
		gl.Vertex3f(-sunSize, 100.0, sunSize)
		gl.End()
	}

	if a.texMoonPhases != nil {
		const moonSize = float32(20.0)
		phase := moonPhase(snap.WorldTime)
		phaseX := phase % 4
		phaseY := (phase / 4) % 2
		u0 := float32(phaseX) / 4.0
		v0 := float32(phaseY) / 2.0
		u1 := float32(phaseX+1) / 4.0
		v1 := float32(phaseY+1) / 2.0

		a.texMoonPhases.bind()
		gl.Begin(gl.QUADS)
		gl.TexCoord2f(u1, v1)
		gl.Vertex3f(-moonSize, -100.0, moonSize)
		gl.TexCoord2f(u0, v1)
		gl.Vertex3f(moonSize, -100.0, moonSize)
		gl.TexCoord2f(u0, v0)
		gl.Vertex3f(moonSize, -100.0, -moonSize)
		gl.TexCoord2f(u1, v0)
		gl.Vertex3f(-moonSize, -100.0, -moonSize)
		gl.End()
	}

	gl.Disable(gl.TEXTURE_2D)
	if star > 0.0 && a.skyStarListID != 0 {
		gl.Color4f(star, star, star, star)
		gl.CallList(a.skyStarListID)
	}
	gl.Color4f(1.0, 1.0, 1.0, 1.0)

	gl.PopMatrix()
	gl.Disable(gl.BLEND)
	gl.Disable(gl.TEXTURE_2D)
	if a.skyList2ID != 0 {
		_, camY, _ := a.interpolatedRenderPlayer(float64(partial), snap)
		const horizon = 63.0
		dy := camY - horizon
		gl.Color3f(skyR*0.2+0.04, skyG*0.2+0.04, skyB*0.6+0.1)
		gl.PushMatrix()
		gl.Translatef(0.0, float32(-(dy - 16.0)), 0.0)
		gl.CallList(a.skyList2ID)
		gl.PopMatrix()
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.ALPHA_TEST)
	gl.Color4f(1.0, 1.0, 1.0, 1.0)
	gl.PopMatrix()
}

// Translation reference:
// - net.minecraft.src.RenderGlobal.RenderGlobal(...) sky/star display lists
// - net.minecraft.src.RenderGlobal.renderStars()
func (a *App) initSkyRenderLists() {
	a.releaseSkyRenderLists()
	base := gl.GenLists(3)
	if base == 0 {
		return
	}
	a.skyStarListID = base
	a.skyListID = base + 1
	a.skyList2ID = base + 2

	gl.NewList(a.skyStarListID, gl.COMPILE)
	renderSkyStars()
	gl.EndList()

	gl.NewList(a.skyListID, gl.COMPILE)
	drawSkyGridPlane(16.0, false)
	gl.EndList()

	gl.NewList(a.skyList2ID, gl.COMPILE)
	drawSkyGridPlane(-16.0, true)
	gl.EndList()
}

func drawSkyGridPlane(y float32, reverse bool) {
	const step = 64
	maxRange := step * (256/step + 2)
	gl.Begin(gl.QUADS)
	for x := -maxRange; x <= maxRange; x += step {
		for z := -maxRange; z <= maxRange; z += step {
			x0 := float32(x)
			x1 := float32(x + step)
			z0 := float32(z)
			z1 := float32(z + step)
			if reverse {
				gl.Vertex3f(x1, y, z0)
				gl.Vertex3f(x0, y, z0)
				gl.Vertex3f(x0, y, z1)
				gl.Vertex3f(x1, y, z1)
			} else {
				gl.Vertex3f(x0, y, z0)
				gl.Vertex3f(x1, y, z0)
				gl.Vertex3f(x1, y, z1)
				gl.Vertex3f(x0, y, z1)
			}
		}
	}
	gl.End()
}

func renderSkyStars() {
	r := rand.New(rand.NewSource(10842))
	gl.Begin(gl.QUADS)
	for i := 0; i < 1500; i++ {
		x := r.Float64()*2.0 - 1.0
		y := r.Float64()*2.0 - 1.0
		z := r.Float64()*2.0 - 1.0
		size := 0.15 + r.Float64()*0.1
		lenSq := x*x + y*y + z*z
		if lenSq >= 1.0 || lenSq <= 0.01 {
			continue
		}

		invLen := 1.0 / math.Sqrt(lenSq)
		x *= invLen
		y *= invLen
		z *= invLen
		cx := x * 100.0
		cy := y * 100.0
		cz := z * 100.0

		yaw := math.Atan2(x, z)
		sinYaw := math.Sin(yaw)
		cosYaw := math.Cos(yaw)
		pitch := math.Atan2(math.Sqrt(x*x+z*z), y)
		sinPitch := math.Sin(pitch)
		cosPitch := math.Cos(pitch)
		rot := r.Float64() * math.Pi * 2.0
		sinRot := math.Sin(rot)
		cosRot := math.Cos(rot)

		for v := 0; v < 4; v++ {
			vx := float64((v&2)-1) * size
			vz := float64(((v+1)&2)-1) * size
			tx := vx*cosRot - vz*sinRot
			tz := vz*cosRot + vx*sinRot
			ty := tx*sinPitch + 0*cosPitch
			t2x := 0*sinPitch - tx*cosPitch
			rx := t2x*sinYaw - tz*cosYaw
			rz := tz*sinYaw + t2x*cosYaw
			gl.Vertex3d(cx+rx, cy+ty, cz+rz)
		}
	}
	gl.End()
}

func (a *App) drawSkyGradientDome(worldTime int64, partial float32) {
	skyR, skyG, skyB := worldSkyColor(worldTime, partial, 0.8)
	fogR, fogG, fogB := worldFogColor(worldTime, partial)

	gl.Disable(gl.TEXTURE_2D)
	gl.ShadeModel(gl.SMOOTH)
	gl.Begin(gl.TRIANGLE_FAN)
	gl.Color4f(skyR, skyG, skyB, 1.0)
	gl.Vertex3f(0.0, 120.0, 0.0)
	const slices = 48
	const radius = float32(300.0)
	gl.Color4f(fogR, fogG, fogB, 1.0)
	for i := 0; i <= slices; i++ {
		ang := float64(i) * math.Pi * 2.0 / float64(slices)
		x := float32(math.Cos(ang)) * radius
		z := float32(math.Sin(ang)) * radius
		gl.Vertex3f(x, 0.0, z)
	}
	gl.End()
	gl.ShadeModel(gl.FLAT)
	gl.Enable(gl.TEXTURE_2D)
}

// Translation reference:
// - net.minecraft.src.GameSettings.shouldRenderClouds()
func (a *App) shouldRenderClouds() bool {
	return a.cloudsEnabled && a.renderDistance < 2
}

func (a *App) drawCloudLayer(snap netclient.StateSnapshot, partial float32) {
	if a.fancyGraphics {
		a.drawCloudLayerFancy(snap, partial)
		return
	}
	a.drawCloudLayerFast(snap, partial)
}

// Translation reference:
// - net.minecraft.src.RenderGlobal.renderCloudsFancy(...)
func (a *App) drawCloudLayerFancy(snap netclient.StateSnapshot, partial float32) {
	if a.texClouds == nil {
		return
	}

	camX, camY, camZ := a.interpolatedRenderPlayer(float64(partial), snap)
	cloudR, cloudG, cloudB := cloudColor(snap.WorldTime, partial)
	const (
		scaleXZ      = float32(12.0)
		layerHeight  = float32(4.0)
		tileSize     = 8
		radiusTiles  = 4
		eps          = float32(9.765625e-4) // 1/1024
		uvScale      = float32(0.00390625)  // 1/256
		scrollFactor = 0.029999999329447746
	)
	cloudAnim := float64(snap.WorldAge) + float64(partial)
	px := (camX + cloudAnim*scrollFactor) / float64(scaleXZ)
	pz := (camZ / float64(scaleXZ)) + 0.33000001311302185
	cloudY := float32(128.0-camY) + 0.33
	sectionX := math.Floor(px / 2048.0)
	sectionZ := math.Floor(pz / 2048.0)
	px -= sectionX * 2048.0
	pz -= sectionZ * 2048.0

	uBase := float32(math.Floor(px)) * uvScale
	vBase := float32(math.Floor(pz)) * uvScale
	xFrac := float32(px - math.Floor(px))
	zFrac := float32(pz - math.Floor(pz))

	gl.PushMatrix()
	gl.Scalef(scaleXZ, 1.0, scaleXZ)
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	a.texClouds.bind()

	for pass := 0; pass < 2; pass++ {
		if pass == 0 {
			gl.ColorMask(false, false, false, false)
		} else {
			gl.ColorMask(true, true, true, true)
		}

		for gridX := -radiusTiles + 1; gridX <= radiusTiles; gridX++ {
			for gridZ := -radiusTiles + 1; gridZ <= radiusTiles; gridZ++ {
				pxTile := float32(gridX * tileSize)
				pzTile := float32(gridZ * tileSize)
				x0 := pxTile - xFrac
				z0 := pzTile - zFrac
				x1 := x0 + float32(tileSize)
				z1 := z0 + float32(tileSize)
				u0 := pxTile*uvScale + uBase
				v0 := pzTile*uvScale + vBase
				u1 := (pxTile+float32(tileSize))*uvScale + uBase
				v1 := (pzTile+float32(tileSize))*uvScale + vBase

				gl.Begin(gl.QUADS)

				if cloudY > -layerHeight-1.0 {
					gl.Color4f(cloudR*0.7, cloudG*0.7, cloudB*0.7, 0.8)
					gl.Normal3f(0.0, -1.0, 0.0)
					gl.TexCoord2f(u0, v1)
					gl.Vertex3f(x0, cloudY, z1)
					gl.TexCoord2f(u1, v1)
					gl.Vertex3f(x1, cloudY, z1)
					gl.TexCoord2f(u1, v0)
					gl.Vertex3f(x1, cloudY, z0)
					gl.TexCoord2f(u0, v0)
					gl.Vertex3f(x0, cloudY, z0)
				}

				if cloudY <= layerHeight+1.0 {
					gl.Color4f(cloudR, cloudG, cloudB, 0.8)
					gl.Normal3f(0.0, 1.0, 0.0)
					gl.TexCoord2f(u0, v1)
					gl.Vertex3f(x0, cloudY+layerHeight-eps, z1)
					gl.TexCoord2f(u1, v1)
					gl.Vertex3f(x1, cloudY+layerHeight-eps, z1)
					gl.TexCoord2f(u1, v0)
					gl.Vertex3f(x1, cloudY+layerHeight-eps, z0)
					gl.TexCoord2f(u0, v0)
					gl.Vertex3f(x0, cloudY+layerHeight-eps, z0)
				}

				gl.Color4f(cloudR*0.9, cloudG*0.9, cloudB*0.9, 0.8)
				if gridX > -1 {
					gl.Normal3f(-1.0, 0.0, 0.0)
					for i := 0; i < tileSize; i++ {
						xs := x0 + float32(i)
						u := (pxTile+float32(i)+0.5)*uvScale + uBase
						gl.TexCoord2f(u, v1)
						gl.Vertex3f(xs, cloudY, z1)
						gl.TexCoord2f(u, v1)
						gl.Vertex3f(xs, cloudY+layerHeight, z1)
						gl.TexCoord2f(u, v0)
						gl.Vertex3f(xs, cloudY+layerHeight, z0)
						gl.TexCoord2f(u, v0)
						gl.Vertex3f(xs, cloudY, z0)
					}
				}
				if gridX <= 1 {
					gl.Normal3f(1.0, 0.0, 0.0)
					for i := 0; i < tileSize; i++ {
						xs := x0 + float32(i) + 1.0 - eps
						u := (pxTile+float32(i)+0.5)*uvScale + uBase
						gl.TexCoord2f(u, v1)
						gl.Vertex3f(xs, cloudY, z1)
						gl.TexCoord2f(u, v1)
						gl.Vertex3f(xs, cloudY+layerHeight, z1)
						gl.TexCoord2f(u, v0)
						gl.Vertex3f(xs, cloudY+layerHeight, z0)
						gl.TexCoord2f(u, v0)
						gl.Vertex3f(xs, cloudY, z0)
					}
				}

				gl.Color4f(cloudR*0.8, cloudG*0.8, cloudB*0.8, 0.8)
				if gridZ > -1 {
					gl.Normal3f(0.0, 0.0, -1.0)
					for i := 0; i < tileSize; i++ {
						zs := z0 + float32(i)
						v := (pzTile+float32(i)+0.5)*uvScale + vBase
						gl.TexCoord2f(u0, v)
						gl.Vertex3f(x0, cloudY+layerHeight, zs)
						gl.TexCoord2f(u1, v)
						gl.Vertex3f(x1, cloudY+layerHeight, zs)
						gl.TexCoord2f(u1, v)
						gl.Vertex3f(x1, cloudY, zs)
						gl.TexCoord2f(u0, v)
						gl.Vertex3f(x0, cloudY, zs)
					}
				}
				if gridZ <= 1 {
					gl.Normal3f(0.0, 0.0, 1.0)
					for i := 0; i < tileSize; i++ {
						zs := z0 + float32(i) + 1.0 - eps
						v := (pzTile+float32(i)+0.5)*uvScale + vBase
						gl.TexCoord2f(u0, v)
						gl.Vertex3f(x0, cloudY+layerHeight, zs)
						gl.TexCoord2f(u1, v)
						gl.Vertex3f(x1, cloudY+layerHeight, zs)
						gl.TexCoord2f(u1, v)
						gl.Vertex3f(x1, cloudY, zs)
						gl.TexCoord2f(u0, v)
						gl.Vertex3f(x0, cloudY, zs)
					}
				}

				gl.End()
			}
		}
	}

	gl.ColorMask(true, true, true, true)
	gl.PopMatrix()
	gl.Color4f(1.0, 1.0, 1.0, 1.0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
}

// Translation reference:
// - net.minecraft.src.RenderGlobal.renderClouds(...)
func (a *App) drawCloudLayerFast(snap netclient.StateSnapshot, partial float32) {
	if a.texClouds == nil {
		return
	}
	camX, camY, camZ := a.interpolatedRenderPlayer(float64(partial), snap)
	cloudR, cloudG, cloudB := cloudColor(snap.WorldTime, partial)
	const (
		tileSize     = 32
		radius       = 256
		uvScale      = float32(4.8828125e-4) // 1/2048
		scrollFactor = 0.029999999329447746
	)
	cloudAnim := float64(snap.WorldAge) + float64(partial)
	px := camX + cloudAnim*scrollFactor
	pz := camZ
	sectionX := math.Floor(px / 2048.0)
	sectionZ := math.Floor(pz / 2048.0)
	px -= sectionX * 2048.0
	pz -= sectionZ * 2048.0
	cloudY := float32(128.0-camY) + 0.33

	uBase := float32(px) * uvScale
	vBase := float32(pz) * uvScale

	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	a.texClouds.bind()
	gl.Color4f(cloudR, cloudG, cloudB, 0.8)
	gl.Begin(gl.QUADS)
	for x := -radius; x < radius; x += tileSize {
		for z := -radius; z < radius; z += tileSize {
			x0 := float32(x)
			z0 := float32(z)
			x1 := x0 + tileSize
			z1 := z0 + tileSize
			u0 := x0*uvScale + uBase
			v0 := z0*uvScale + vBase
			u1 := x1*uvScale + uBase
			v1 := z1*uvScale + vBase
			gl.TexCoord2f(u0, v1)
			gl.Vertex3f(x0, cloudY, z1)
			gl.TexCoord2f(u1, v1)
			gl.Vertex3f(x1, cloudY, z1)
			gl.TexCoord2f(u1, v0)
			gl.Vertex3f(x1, cloudY, z0)
			gl.TexCoord2f(u0, v0)
			gl.Vertex3f(x0, cloudY, z0)
		}
	}
	gl.End()
	gl.Color4f(1.0, 1.0, 1.0, 1.0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
}

func celestialAngle(worldTime int64, partial float32) float32 {
	tickOfDay := int(worldTime % 24000)
	angle := (float32(tickOfDay)+partial)/24000.0 - 0.25
	if angle < 0.0 {
		angle++
	}
	if angle > 1.0 {
		angle--
	}
	base := angle
	angle = 1.0 - float32((math.Cos(float64(angle)*math.Pi)+1.0)*0.5)
	angle = base + (angle-base)/3.0
	return angle
}

func worldSkyColor(worldTime int64, partial, biomeTemp float32) (float32, float32, float32) {
	angle := celestialAngle(worldTime, partial)
	dayLight := float32(math.Cos(float64(angle)*math.Pi*2.0))*2.0 + 0.5
	dayLight = clampFloat32(dayLight, 0.0, 1.0)

	r, g, b := skyColorByTemperature(biomeTemp)
	return r * dayLight, g * dayLight, b * dayLight
}

func worldFogColor(worldTime int64, partial float32) (float32, float32, float32) {
	angle := celestialAngle(worldTime, partial)
	f := float32(math.Cos(float64(angle)*math.Pi*2.0))*2.0 + 0.5
	f = clampFloat32(f, 0.0, 1.0)
	// Translation reference:
	// - net.minecraft.src.WorldProvider.getFogColor(...)
	r := float32(0.7529412) * (f*0.94 + 0.06)
	g := float32(0.84705883) * (f*0.94 + 0.06)
	b := float32(1.0) * (f*0.91 + 0.09)
	return r, g, b
}

func starBrightness(worldTime int64, partial float32) float32 {
	angle := celestialAngle(worldTime, partial)
	v := 1.0 - (float32(math.Cos(float64(angle)*math.Pi*2.0))*2.0 + 0.25)
	v = clampFloat32(v, 0.0, 1.0)
	return v * v * 0.5
}

func cloudColor(worldTime int64, partial float32) (float32, float32, float32) {
	angle := celestialAngle(worldTime, partial)
	f := float32(math.Cos(float64(angle)*math.Pi*2.0))*2.0 + 0.5
	f = clampFloat32(f, 0.0, 1.0)
	// Translation reference:
	// - net.minecraft.src.World.getCloudColour(...)
	r := float32(1.0) * (f*0.9 + 0.1)
	g := float32(1.0) * (f*0.9 + 0.1)
	b := float32(1.0) * (f*0.85 + 0.15)
	return r, g, b
}

func skyColorByTemperature(temp float32) (float32, float32, float32) {
	v := temp / 3.0
	v = clampFloat32(v, -1.0, 1.0)
	// Translation reference:
	// - net.minecraft.src.BiomeGenBase.getSkyColorByTemp(...)
	//   HSB(0.62222224F - v*0.05F, 0.5F + v*0.1F, 1.0F)
	return hsbToRGB(0.62222224-v*0.05, 0.5+v*0.1, 1.0)
}

func sunriseSunsetColors(celestial float32) (float32, float32, float32, float32, bool) {
	// Translation reference:
	// - net.minecraft.src.WorldProvider.calcSunriseSunsetColors(...)
	c := float32(math.Cos(float64(celestial) * math.Pi * 2.0))
	const spread = float32(0.4)
	if c < -spread || c > spread {
		return 0, 0, 0, 0, false
	}
	t := c/spread*0.5 + 0.5
	alpha := 1.0 - (1.0-float32(math.Sin(float64(t)*math.Pi)))*0.99
	alpha *= alpha
	r := t*0.3 + 0.7
	g := t*t*0.7 + 0.2
	b := t*t*0.0 + 0.2
	return r, g, b, alpha, true
}

func moonPhase(worldTime int64) int {
	phase := int((worldTime / 24000) % 8)
	if phase < 0 {
		phase += 8
	}
	return phase
}

func hsbToRGB(h, s, v float32) (float32, float32, float32) {
	if s <= 0 {
		return v, v, v
	}
	h = float32(math.Mod(float64(h), 1.0))
	if h < 0 {
		h += 1.0
	}
	sector := h * 6.0
	i := int(math.Floor(float64(sector)))
	f := sector - float32(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))
	switch i % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

func clampFloat32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *App) currentFOVDegrees() float64 {
	return 70.0 + clampFloat64(a.fovSetting, 0.0, 1.0)*40.0
}

func (a *App) optionFOVLabel() string {
	fov := a.currentFOVDegrees()
	if math.Abs(fov-70.0) < 1.0e-6 {
		return "FOV: Normal"
	}
	return fmt.Sprintf("FOV: %d", int(math.Round(fov)))
}

func (a *App) sensitivityPercent() int {
	v := int(a.mouseSens * 200.0)
	if v < 0 {
		return 0
	}
	if v > 200 {
		return 200
	}
	return v
}

// Translation reference:
// - net.minecraft.src.EntityRenderer.updateCameraAndRender(float)
// - net.minecraft.src.Entity.setAngles(float,float)
func (a *App) mouseTurnScale() float64 {
	sens := clampFloat64(a.mouseSens, 0.0, 1.0)*0.6 + 0.2
	return sens * sens * sens * 8.0 * 0.15
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func normalizeRenderDistanceMode(mode int) int {
	return clampInt(mode, 0, 3)
}

func renderDistanceChunksToMode(chunks int) int {
	type choice struct {
		mode   int
		chunks int
	}
	choices := []choice{
		{mode: 0, chunks: 16},
		{mode: 1, chunks: 8},
		{mode: 2, chunks: 4},
		{mode: 3, chunks: 2},
	}
	if chunks <= 0 {
		return 1
	}
	bestMode := choices[0].mode
	bestDiff := int(math.Abs(float64(chunks - choices[0].chunks)))
	for _, c := range choices[1:] {
		diff := int(math.Abs(float64(chunks - c.chunks)))
		if diff < bestDiff {
			bestDiff = diff
			bestMode = c.mode
		}
	}
	return bestMode
}

func renderDistanceModeToChunks(mode int) int {
	switch normalizeRenderDistanceMode(mode) {
	case 0:
		return 16
	case 1:
		return 8
	case 2:
		return 4
	default:
		return 2
	}
}

func renderDistanceModeName(mode int) string {
	return renderDistanceModeNames[normalizeRenderDistanceMode(mode)]
}

func (a *App) renderDistanceChunks() int {
	return renderDistanceModeToChunks(a.renderDistance)
}

func (a *App) renderDistanceModeName() string {
	return renderDistanceModeName(a.renderDistance)
}

func (a *App) optionRenderDistanceLabel() string {
	return "Render Distance: " + a.renderDistanceModeName()
}

func (a *App) optionFramerateLabel() string {
	mode := clampInt(a.limitFramerateMode, 0, len(framerateModeNames)-1)
	return "Max Framerate: " + framerateModeNames[mode]
}

func (a *App) optionGUIScaleLabel() string {
	mode := clampInt(a.guiScaleMode, 0, len(guiScaleModeNames)-1)
	return "GUI Scale: " + guiScaleModeNames[mode]
}

func (a *App) optionInvertMouseLabel() string {
	if a.invertMouse {
		return "Invert Mouse: ON"
	}
	return "Invert Mouse: OFF"
}

func (a *App) optionGraphicsLabel() string {
	if a.fancyGraphics {
		return "Graphics: Fancy"
	}
	return "Graphics: Fast"
}

func (a *App) optionCloudsLabel() string {
	if a.cloudsEnabled {
		return "Clouds: ON"
	}
	return "Clouds: OFF"
}

func volumePercent(v float64) int {
	p := int(math.Round(clampFloat64(v, 0.0, 1.0) * 100.0))
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

func (a *App) optionMusicVolumeLabel() string {
	p := volumePercent(a.musicVolume)
	if p <= 0 {
		return "Music: OFF"
	}
	return fmt.Sprintf("Music: %d%%", p)
}

func (a *App) optionSoundVolumeLabel() string {
	p := volumePercent(a.soundVolume)
	if p <= 0 {
		return "Sound: OFF"
	}
	return fmt.Sprintf("Sound: %d%%", p)
}

func (a *App) drawWorld(snap netclient.StateSnapshot) {
	a.renderFrame++
	centerChunkX := int32(int(math.Floor(snap.PlayerX)) >> 4)
	centerChunkZ := int32(int(math.Floor(snap.PlayerZ)) >> 4)
	radius := a.renderDistanceChunks()
	if radius < 1 {
		radius = 1
	}
	radiusSq := radius * radius
	buildBudget := maxChunkBuilds
	type translucentChunkDraw struct {
		listID uint32
		distSq float64
	}
	translucentChunks := make([]translucentChunkDraw, 0, (radius*2+1)*(radius*2+1))

	for cx := int(centerChunkX) - radius; cx <= int(centerChunkX)+radius; cx++ {
		for cz := int(centerChunkZ) - radius; cz <= int(centerChunkZ)+radius; cz++ {
			dx := cx - int(centerChunkX)
			dz := cz - int(centerChunkZ)
			if dx*dx+dz*dz > radiusSq {
				continue
			}
			chunkX := int32(cx)
			chunkZ := int32(cz)
			key := chunk.NewCoordIntPair(chunkX, chunkZ)
			baseRevision := a.session.ChunkRevision(chunkX, chunkZ)
			if baseRevision == 0 {
				// Chunk not available in client cache yet.
				continue
			}
			revision := chunkMeshRevision(chunkX, chunkZ, func(rx, rz int32) uint64 {
				return a.session.ChunkRevision(rx, rz)
			})
			entry, ok := a.chunkRenderCache[key]
			if !ok {
				entry = &chunkRenderEntry{}
				a.chunkRenderCache[key] = entry
			}

			if (entry.listOpaque == 0 || entry.revision != revision) && buildBudget > 0 {
				a.rebuildChunkRenderEntry(entry, chunkX, chunkZ, revision)
				buildBudget--
			}

			if entry.listOpaque != 0 {
				gl.CallList(entry.listOpaque)
			}
			if entry.listTranslucent != 0 {
				chunkCenterX := float64(int(chunkX)<<4) + 8.0
				chunkCenterZ := float64(int(chunkZ)<<4) + 8.0
				dx := chunkCenterX - snap.PlayerX
				dz := chunkCenterZ - snap.PlayerZ
				translucentChunks = append(translucentChunks, translucentChunkDraw{
					listID: entry.listTranslucent,
					distSq: dx*dx + dz*dz,
				})
			}
			if entry.listOpaque != 0 || entry.listTranslucent != 0 {
				entry.lastUsedFrame = a.renderFrame
			}
		}
	}

	if len(translucentChunks) > 1 {
		sort.Slice(translucentChunks, func(i, j int) bool {
			return translucentChunks[i].distSq > translucentChunks[j].distSq
		})
	}
	if len(translucentChunks) > 0 {
		// Translation reference:
		// - net.minecraft.src.EntityRenderer / RenderGlobal translucent pass.
		// Blend pass keeps depth test but disables depth writes and renders back-to-front.
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.DepthMask(false)
		for _, draw := range translucentChunks {
			gl.CallList(draw.listID)
		}
		gl.DepthMask(true)
		gl.Disable(gl.BLEND)
		gl.Color4f(1, 1, 1, 1)
	}
	a.cleanupChunkRenderCache()
}

func chunkMeshRevision(chunkX, chunkZ int32, revisionAt func(x, z int32) uint64) uint64 {
	if revisionAt == nil {
		return 0
	}
	base := revisionAt(chunkX, chunkZ)
	if base == 0 {
		return 0
	}

	// Include neighbor chunk revisions so border-dependent mesh results
	// (face culling, liquid corner heights) rebuild when adjacent chunks load/unload.
	mix := uint64(1469598103934665603)
	for dz := int32(-1); dz <= 1; dz++ {
		for dx := int32(-1); dx <= 1; dx++ {
			rev := revisionAt(chunkX+dx, chunkZ+dz)
			coord := uint64(uint32((dx+1)&3)<<2 | uint32((dz+1)&3))
			v := rev ^ (coord * 0x9E3779B185EBCA87)
			mix ^= v + 0x9E3779B185EBCA87 + (mix << 6) + (mix >> 2)
		}
	}
	return mix
}

func (a *App) rebuildChunkRenderEntry(entry *chunkRenderEntry, chunkX, chunkZ int32, revision uint64) {
	if entry.listOpaque != 0 {
		gl.DeleteLists(entry.listOpaque, 1)
		entry.listOpaque = 0
	}
	if entry.listTranslucent != 0 {
		gl.DeleteLists(entry.listTranslucent, 1)
		entry.listTranslucent = 0
	}

	listOpaque := gl.GenLists(1)
	if listOpaque == 0 {
		return
	}

	solidBatches := make(map[uint32]*textureBatch, 24)
	translucentBatches := make(map[uint32]*textureBatch, 8)
	overlayBatches := make(map[uint32]*textureBatch, 2)

	baseX := int(chunkX) << 4
	baseZ := int(chunkZ) << 4
	for localX := 0; localX < 16; localX++ {
		worldX := baseX + localX
		for localZ := 0; localZ < 16; localZ++ {
			worldZ := baseZ + localZ
			topY := -1
			for y := 255; y >= 0; y-- {
				id, _, ok := a.session.BlockAt(worldX, y, worldZ)
				if ok && id != 0 {
					topY = y
					break
				}
			}
			if topY < 0 {
				continue
			}
			for y := 0; y <= topY; y++ {
				id, meta, ok := a.session.BlockAt(worldX, y, worldZ)
				if !ok || id == 0 {
					continue
				}
				faces := a.visibleBlockFaces(worldX, y, worldZ, id, meta)
				if !faces.any() && !isCrossedPlantRenderBlock(id) && !isFlatPlantRenderBlock(id) && id != 106 {
					continue
				}
				if a.appendTexturedBlockBatches(float32(worldX), float32(y), float32(worldZ), id, meta, faces, solidBatches, translucentBatches, overlayBatches) {
					continue
				}
				r, g, b := colorForBlock(id)
				a.appendFallbackBlockBatches(float32(worldX), float32(y), float32(worldZ), r, g, b, faces, solidBatches)
			}
		}
	}

	gl.NewList(listOpaque, gl.COMPILE)
	a.emitTextureBatches(solidBatches)
	if len(overlayBatches) > 0 {
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		a.emitTextureBatches(overlayBatches)
		gl.Disable(gl.BLEND)
	}
	gl.Color4f(1, 1, 1, 1)
	gl.EndList()

	var listTranslucent uint32
	if len(translucentBatches) > 0 {
		listTranslucent = gl.GenLists(1)
		if listTranslucent != 0 {
			gl.NewList(listTranslucent, gl.COMPILE)
			a.emitTextureBatches(translucentBatches)
			gl.Color4f(1, 1, 1, 1)
			gl.EndList()
		}
	}

	entry.listOpaque = listOpaque
	entry.listTranslucent = listTranslucent
	entry.revision = revision
	entry.lastUsedFrame = a.renderFrame
}

func (a *App) cleanupChunkRenderCache() {
	const staleFrames = 600
	if a.renderFrame%60 != 0 {
		return
	}
	for key, entry := range a.chunkRenderCache {
		if entry == nil {
			delete(a.chunkRenderCache, key)
			continue
		}
		if a.renderFrame-entry.lastUsedFrame <= staleFrames {
			continue
		}
		if entry.listOpaque != 0 {
			gl.DeleteLists(entry.listOpaque, 1)
		}
		if entry.listTranslucent != 0 {
			gl.DeleteLists(entry.listTranslucent, 1)
		}
		delete(a.chunkRenderCache, key)
	}
}

func (a *App) visibleBlockFaces(x, y, z, currentID, currentMeta int) visibleFaces {
	return visibleFaces{
		Down:  a.shouldRenderFace(currentID, currentMeta, faceDown, x, y-1, z),
		Up:    a.shouldRenderFace(currentID, currentMeta, faceUp, x, y+1, z),
		North: a.shouldRenderFace(currentID, currentMeta, faceNorth, x, y, z-1),
		South: a.shouldRenderFace(currentID, currentMeta, faceSouth, x, y, z+1),
		West:  a.shouldRenderFace(currentID, currentMeta, faceWest, x-1, y, z),
		East:  a.shouldRenderFace(currentID, currentMeta, faceEast, x+1, y, z),
	}
}

func (a *App) shouldRenderFace(currentID, currentMeta, face, nx, ny, nz int) bool {
	neighborID, neighborMeta, ok := a.session.BlockAt(nx, ny, nz)
	if !ok || neighborID == 0 {
		return true
	}
	if isSameLiquidMaterial(currentID, neighborID) {
		return false
	}
	if neighborID == currentID {
		// Translation reference:
		// - net.minecraft.src.BlockLeavesBase.shouldSideBeRendered(...)
		// Fancy leaves render inner shared faces for denser canopy look.
		if currentID == 18 {
			return a.fancyGraphics
		}
	}
	if !isOpaqueRenderBlock(neighborID) {
		return true
	}

	currentBounds := blockRenderRelativeAABB(currentID, currentMeta)
	neighborBounds := blockRenderRelativeAABB(neighborID, neighborMeta)
	if !faceFullyOccludedByNeighbor(face, currentBounds, neighborBounds) {
		return true
	}
	return false
}

func (a *App) appendTexturedBlockBatches(
	x, y, z float32,
	blockID int,
	blockMeta int,
	faces visibleFaces,
	solidBatches map[uint32]*textureBatch,
	translucentBatches map[uint32]*textureBatch,
	overlayBatches map[uint32]*textureBatch,
) bool {
	if len(a.blockTextureDefs) == 0 || len(a.blockTextures) == 0 {
		return false
	}
	if _, ok := a.blockTextureDefs[blockID]; !ok {
		return false
	}
	blockX, blockY, blockZ := int(x), int(y), int(z)
	if isLiquidRenderBlock(blockID) {
		return a.appendLiquidBlockBatches(x, y, z, blockX, blockY, blockZ, blockID, blockMeta, translucentBatches)
	}
	if blockID == 106 {
		return a.appendVineBatches(x, y, z, blockX, blockY, blockZ, blockMeta, solidBatches)
	}
	if isFlatPlantRenderBlock(blockID) {
		return a.appendFlatPlantBatches(x, y, z, blockX, blockY, blockZ, blockID, solidBatches)
	}
	if isCrossedPlantRenderBlock(blockID) {
		return a.appendCrossedPlantBatches(x, y, z, blockX, blockY, blockZ, blockID, blockMeta, solidBatches)
	}

	top := a.blockTextureForFaceMeta(blockID, blockMeta, faceUp)
	bottom := a.blockTextureForFaceMeta(blockID, blockMeta, faceDown)
	north := a.blockTextureForFaceMeta(blockID, blockMeta, faceNorth)
	south := a.blockTextureForFaceMeta(blockID, blockMeta, faceSouth)
	west := a.blockTextureForFaceMeta(blockID, blockMeta, faceWest)
	east := a.blockTextureForFaceMeta(blockID, blockMeta, faceEast)

	if (faces.Up && top == nil) ||
		(faces.Down && bottom == nil) ||
		(faces.North && north == nil) ||
		(faces.South && south == nil) ||
		(faces.West && west == nil) ||
		(faces.East && east == nil) {
		return false
	}

	shape := blockRenderRelativeAABB(blockID, blockMeta)
	x1 := x + float32(shape.minX)
	y1 := y + float32(shape.minY)
	z1 := z + float32(shape.minZ)
	x2 := x + float32(shape.maxX)
	y2 := y + float32(shape.maxY)
	z2 := z + float32(shape.maxZ)
	addFace := func(tex *texture2D, shade, tintR, tintG, tintB float32, v0, v1, v2, v3 [3]float32, target map[uint32]*textureBatch) {
		b := ensureTextureBatch(target, tex)
		if b == nil {
			return
		}
		appendTexturedQuad(
			b,
			tintR*shade, tintG*shade, tintB*shade,
			v0, v1, v2, v3,
		)
	}
	addMainFace := func(tex *texture2D, face int, shade float32, v0, v1, v2, v3 [3]float32) {
		tintR, tintG, tintB := a.blockFaceTintAt(blockX, blockY, blockZ, blockID, blockMeta, face)
		addFace(tex, shade, tintR, tintG, tintB, v0, v1, v2, v3, solidBatches)
	}

	if faces.Up {
		addMainFace(top, faceUp, 1.0,
			[3]float32{x1, y2, z1},
			[3]float32{x1, y2, z2},
			[3]float32{x2, y2, z2},
			[3]float32{x2, y2, z1},
		)
	}
	if faces.Down {
		addMainFace(bottom, faceDown, 0.52,
			[3]float32{x1, y1, z1},
			[3]float32{x2, y1, z1},
			[3]float32{x2, y1, z2},
			[3]float32{x1, y1, z2},
		)
	}
	if faces.North {
		addMainFace(north, faceNorth, 0.80,
			[3]float32{x2, y1, z1},
			[3]float32{x1, y1, z1},
			[3]float32{x1, y2, z1},
			[3]float32{x2, y2, z1},
		)
	}
	if faces.South {
		addMainFace(south, faceSouth, 0.80,
			[3]float32{x1, y1, z2},
			[3]float32{x2, y1, z2},
			[3]float32{x2, y2, z2},
			[3]float32{x1, y2, z2},
		)
	}
	if faces.West {
		addMainFace(west, faceWest, 0.65,
			[3]float32{x1, y1, z1},
			[3]float32{x1, y1, z2},
			[3]float32{x1, y2, z2},
			[3]float32{x1, y2, z1},
		)
	}
	if faces.East {
		addMainFace(east, faceEast, 0.65,
			[3]float32{x2, y1, z2},
			[3]float32{x2, y1, z1},
			[3]float32{x2, y2, z1},
			[3]float32{x2, y2, z2},
		)
	}

	overlay := a.blockSideOverlayTexture(blockID)
	if overlay != nil {
		overlayR, overlayG, overlayB := a.blockSideOverlayTintAt(blockX, blockY, blockZ, blockID)
		if faces.North {
			addFace(overlay, 0.80, overlayR, overlayG, overlayB,
				[3]float32{x2, y1, z1},
				[3]float32{x1, y1, z1},
				[3]float32{x1, y2, z1},
				[3]float32{x2, y2, z1},
				overlayBatches,
			)
		}
		if faces.South {
			addFace(overlay, 0.80, overlayR, overlayG, overlayB,
				[3]float32{x1, y1, z2},
				[3]float32{x2, y1, z2},
				[3]float32{x2, y2, z2},
				[3]float32{x1, y2, z2},
				overlayBatches,
			)
		}
		if faces.West {
			addFace(overlay, 0.65, overlayR, overlayG, overlayB,
				[3]float32{x1, y1, z1},
				[3]float32{x1, y1, z2},
				[3]float32{x1, y2, z2},
				[3]float32{x1, y2, z1},
				overlayBatches,
			)
		}
		if faces.East {
			addFace(overlay, 0.65, overlayR, overlayG, overlayB,
				[3]float32{x2, y1, z2},
				[3]float32{x2, y1, z1},
				[3]float32{x2, y2, z1},
				[3]float32{x2, y2, z2},
				overlayBatches,
			)
		}
	}

	return true
}

// Translation reference:
// - net.minecraft.src.RenderBlocks.renderBlockFluids(...)
// - net.minecraft.src.RenderBlocks.getFluidHeight(...)
// - net.minecraft.src.BlockFluid.getFluidHeightPercent(...)
// - net.minecraft.src.BlockFluid.getFlowDirection(...)
func (a *App) appendLiquidBlockBatches(
	x, y, z float32,
	blockX, blockY, blockZ int,
	blockID int,
	blockMeta int,
	translucentBatches map[uint32]*textureBatch,
) bool {
	if translucentBatches == nil {
		return false
	}

	stillTex := a.blockTextureForFace(blockID, faceUp)
	flowTex := a.blockTextureForFace(blockID, faceNorth)
	if flowTex == nil {
		flowTex = stillTex
	}
	if stillTex == nil && flowTex == nil {
		return false
	}
	if stillTex == nil {
		stillTex = flowTex
	}
	if flowTex == nil {
		flowTex = stillTex
	}

	renderTop := a.shouldRenderLiquidFace(blockID, blockX, blockY+1, blockZ, faceUp)
	renderBottom := a.shouldRenderLiquidFace(blockID, blockX, blockY-1, blockZ, faceDown)
	renderNorth := a.shouldRenderLiquidFace(blockID, blockX, blockY, blockZ-1, faceNorth)
	renderSouth := a.shouldRenderLiquidFace(blockID, blockX, blockY, blockZ+1, faceSouth)
	renderWest := a.shouldRenderLiquidFace(blockID, blockX-1, blockY, blockZ, faceWest)
	renderEast := a.shouldRenderLiquidFace(blockID, blockX+1, blockY, blockZ, faceEast)
	if !renderTop && !renderBottom && !renderNorth && !renderSouth && !renderWest && !renderEast {
		return false
	}

	// Corner heights follow vanilla getFluidHeight sampling order:
	// (x,z), (x,z+1), (x+1,z+1), (x+1,z).
	h00 := a.liquidCornerHeight(blockX, blockY, blockZ, blockID)
	h01 := a.liquidCornerHeight(blockX, blockY, blockZ+1, blockID)
	h11 := a.liquidCornerHeight(blockX+1, blockY, blockZ+1, blockID)
	h10 := a.liquidCornerHeight(blockX+1, blockY, blockZ, blockID)

	tintR, tintG, tintB := a.blockFaceTintAt(blockX, blockY, blockZ, blockID, blockMeta, faceUp)
	const eps = float32(0.0010000000474974513)

	if renderTop {
		tex := stillTex
		flowDirection := a.liquidFlowDirection(blockID, blockX, blockY, blockZ)
		if flowDirection > -999.0 {
			tex = flowTex
		}
		batch := ensureTextureBatch(translucentBatches, tex)
		if batch != nil {
			y00 := y + h00 - eps
			y01 := y + h01 - eps
			y11 := y + h11 - eps
			y10 := y + h10 - eps

			u0, v0 := float32(0), float32(0)
			u1, v1 := float32(0), float32(1)
			u2, v2 := float32(1), float32(1)
			u3, v3 := float32(1), float32(0)
			if flowDirection > -999.0 {
				sinF := float32(math.Sin(flowDirection)) * 0.25
				cosF := float32(math.Cos(flowDirection)) * 0.25
				u0 = 0.5 + (-cosF - sinF)
				v0 = 0.5 + (-cosF + sinF)
				u1 = 0.5 + (-cosF + sinF)
				v1 = 0.5 + (cosF + sinF)
				u2 = 0.5 + (cosF + sinF)
				v2 = 0.5 + (cosF - sinF)
				u3 = 0.5 + (cosF - sinF)
				v3 = 0.5 + (-cosF - sinF)
			}
			u0, v0 = a.liquidAnimatedUV(tex, u0, v0)
			u1, v1 = a.liquidAnimatedUV(tex, u1, v1)
			u2, v2 = a.liquidAnimatedUV(tex, u2, v2)
			u3, v3 = a.liquidAnimatedUV(tex, u3, v3)

			appendTexturedQuadUV(
				batch, tintR, tintG, tintB,
				[3]float32{x + 0, y00, z + 0}, u0, v0,
				[3]float32{x + 0, y01, z + 1}, u1, v1,
				[3]float32{x + 1, y11, z + 1}, u2, v2,
				[3]float32{x + 1, y10, z + 0}, u3, v3,
			)
		}
	}

	if renderBottom {
		batch := ensureTextureBatch(translucentBatches, stillTex)
		if batch != nil {
			y0 := y + eps
			u0, v0 := a.liquidAnimatedUV(stillTex, 0, 0)
			u1, v1 := a.liquidAnimatedUV(stillTex, 1, 0)
			u2, v2 := a.liquidAnimatedUV(stillTex, 1, 1)
			u3, v3 := a.liquidAnimatedUV(stillTex, 0, 1)
			appendTexturedQuadUV(
				batch, tintR*0.5, tintG*0.5, tintB*0.5,
				[3]float32{x + 0, y0, z + 0}, u0, v0,
				[3]float32{x + 1, y0, z + 0}, u1, v1,
				[3]float32{x + 1, y0, z + 1}, u2, v2,
				[3]float32{x + 0, y0, z + 1}, u3, v3,
			)
		}
	}

	sideBatch := ensureTextureBatch(translucentBatches, flowTex)
	if sideBatch == nil {
		return true
	}
	appendSide := func(shade, hA, hB float32, p0, p1, p2, p3 [3]float32) {
		u0, vTopA := a.liquidAnimatedUV(flowTex, 0.0, (1.0-hA)*0.5)
		u1, vTopB := a.liquidAnimatedUV(flowTex, 0.5, (1.0-hB)*0.5)
		_, vBottom := a.liquidAnimatedUV(flowTex, 0.0, 0.5)
		appendTexturedQuadUV(
			sideBatch, tintR*shade, tintG*shade, tintB*shade,
			p0, u0, vTopA,
			p1, u1, vTopB,
			p2, u1, vBottom,
			p3, u0, vBottom,
		)
	}

	if renderNorth {
		appendSide(0.8, h00, h10,
			[3]float32{x + 0, y + h00, z + eps},
			[3]float32{x + 1, y + h10, z + eps},
			[3]float32{x + 1, y + 0, z + eps},
			[3]float32{x + 0, y + 0, z + eps},
		)
	}
	if renderSouth {
		appendSide(0.8, h11, h01,
			[3]float32{x + 1, y + h11, z + 1 - eps},
			[3]float32{x + 0, y + h01, z + 1 - eps},
			[3]float32{x + 0, y + 0, z + 1 - eps},
			[3]float32{x + 1, y + 0, z + 1 - eps},
		)
	}
	if renderWest {
		appendSide(0.6, h01, h00,
			[3]float32{x + eps, y + h01, z + 1},
			[3]float32{x + eps, y + h00, z + 0},
			[3]float32{x + eps, y + 0, z + 0},
			[3]float32{x + eps, y + 0, z + 1},
		)
	}
	if renderEast {
		appendSide(0.6, h10, h11,
			[3]float32{x + 1 - eps, y + h10, z + 0},
			[3]float32{x + 1 - eps, y + h11, z + 1},
			[3]float32{x + 1 - eps, y + 0, z + 1},
			[3]float32{x + 1 - eps, y + 0, z + 0},
		)
	}

	return true
}

func (a *App) shouldRenderLiquidFace(blockID, nx, ny, nz, side int) bool {
	neighborID, _ := a.blockAtOrAir(nx, ny, nz)
	if isSameLiquidMaterial(blockID, neighborID) {
		return false
	}
	if side == faceUp {
		return true
	}
	if neighborID == 79 { // ice special-case from BlockFluid.shouldSideBeRendered
		return false
	}
	return !isOpaqueRenderBlock(neighborID)
}

func (a *App) liquidCornerHeight(x, y, z, blockID int) float32 {
	weight := 0
	accum := float32(0)

	for i := 0; i < 4; i++ {
		nx := x - (i & 1)
		nz := z - ((i >> 1) & 1)
		aboveID, _, aboveOK := a.blockAtOrUnknown(nx, y+1, nz)
		if aboveOK && isSameLiquidMaterial(blockID, aboveID) {
			return 1.0
		}

		id, meta, ok := a.blockAtOrUnknown(nx, y, nz)
		if !ok {
			// Keep border heights stable while neighbor chunks stream in.
			id = blockID
			meta = 0
		}
		if isSameLiquidMaterial(blockID, id) {
			h := fluidHeightPercent(meta)
			if meta >= 8 || meta == 0 {
				accum += h * 10.0
				weight += 10
			}
			accum += h
			weight++
		} else if !block.BlocksMovement(id) {
			accum++
			weight++
		}
	}

	if weight == 0 {
		return 1.0
	}
	return 1.0 - accum/float32(weight)
}

func fluidHeightPercent(meta int) float32 {
	if meta >= 8 {
		meta = 0
	}
	return float32(meta+1) / 9.0
}

func (a *App) liquidAnimatedUV(tex *texture2D, u, v float32) (float32, float32) {
	if tex == nil || tex.Width <= 0 || tex.Height <= tex.Width || tex.Height%tex.Width != 0 {
		return u, v
	}

	frameCount := tex.Height / tex.Width
	if frameCount <= 1 {
		return u, v
	}

	// Translation reference:
	// - net.minecraft.src.TextureWaterFX / TextureWaterFlowFX (tick-driven animation)
	// Use 20 TPS timing to step strip frames.
	frame := int((time.Now().UnixMilli() / 50) % int64(frameCount))
	// Texture strips are loaded flipped with the full image, so frame order is reversed.
	frame = frameCount - 1 - frame
	frameSpan := 1.0 / float32(frameCount)
	return u, float32(frame)*frameSpan + v*frameSpan
}

func (a *App) effectiveLiquidFlowDecay(blockID, x, y, z int) int {
	id, meta := a.blockAtOrAir(x, y, z)
	if !isSameLiquidMaterial(blockID, id) {
		return -1
	}
	if meta >= 8 {
		meta = 0
	}
	return meta
}

func (a *App) liquidFlowDirection(blockID, x, y, z int) float64 {
	flowX, flowZ := a.liquidFlowVector(blockID, x, y, z)
	if math.Abs(flowX) <= 1.0e-9 && math.Abs(flowZ) <= 1.0e-9 {
		return -1000.0
	}
	return math.Atan2(flowZ, flowX) - (math.Pi / 2.0)
}

func (a *App) liquidFlowVector(blockID, x, y, z int) (float64, float64) {
	baseDecay := a.effectiveLiquidFlowDecay(blockID, x, y, z)
	if baseDecay < 0 {
		return 0, 0
	}

	flowX := 0.0
	flowZ := 0.0
	for dir := 0; dir < 4; dir++ {
		nx, nz := x, z
		switch dir {
		case 0:
			nx = x - 1
		case 1:
			nz = z - 1
		case 2:
			nx = x + 1
		case 3:
			nz = z + 1
		}

		neighborDecay := a.effectiveLiquidFlowDecay(blockID, nx, y, nz)
		if neighborDecay < 0 {
			neighborID, _ := a.blockAtOrAir(nx, y, nz)
			if !block.BlocksMovement(neighborID) {
				neighborDecay = a.effectiveLiquidFlowDecay(blockID, nx, y-1, nz)
				if neighborDecay >= 0 {
					delta := neighborDecay - (baseDecay - 8)
					flowX += float64((nx - x) * delta)
					flowZ += float64((nz - z) * delta)
				}
			}
			continue
		}

		delta := neighborDecay - baseDecay
		flowX += float64((nx - x) * delta)
		flowZ += float64((nz - z) * delta)
	}
	return flowX, flowZ
}

func (a *App) blockAtOrAir(x, y, z int) (int, int) {
	id, meta, _ := a.blockAtOrUnknown(x, y, z)
	return id, meta
}

func (a *App) blockAtOrUnknown(x, y, z int) (int, int, bool) {
	if a.session == nil || y < 0 || y >= 256 {
		return 0, 0, false
	}
	id, meta, ok := a.session.BlockAt(x, y, z)
	if !ok {
		return 0, 0, false
	}
	return id, meta, true
}

func (a *App) appendVineBatches(
	x, y, z float32,
	blockX, blockY, blockZ int,
	blockMeta int,
	solidBatches map[uint32]*textureBatch,
) bool {
	tex := a.blockTextureForFace(106, faceUp)
	if tex == nil {
		return false
	}
	renderMeta := blockMeta & 15
	aboveID, aboveMeta, aboveOK := a.session.BlockAt(blockX, blockY+1, blockZ)
	if renderMeta == 0 {
		// Compatibility fallback for invalid meta=0 save states:
		// keep orientation only from same-column vine segments, do not infer from
		// surrounding solid blocks to avoid non-vanilla outward-facing artifacts.
		if aboveOK && aboveID == 106 {
			renderMeta = aboveMeta & 15
		}
		if renderMeta == 0 {
			if belowID, belowMeta, ok := a.session.BlockAt(blockX, blockY-1, blockZ); ok && belowID == 106 && (belowMeta&15) != 0 {
				renderMeta = belowMeta & 15
			}
		}
		if renderMeta == 0 {
			renderMeta = a.inferSingleVineAttachmentMeta(blockX, blockY, blockZ)
		}
	}

	tintR, tintG, tintB := a.blockFaceTintAt(blockX, blockY, blockZ, 106, renderMeta, faceUp)
	batch := ensureTextureBatch(solidBatches, tex)
	if batch == nil {
		return false
	}

	const inset = float32(0.05)
	x0, y0, z0 := x, y, z
	x1, y1, z1 := x+1, y+1, z+1

	// Translation reference:
	// - net.minecraft.src.RenderBlocks.renderBlockVine(...)
	appendVineDoubleSidedA := func(v0, v1, v2, v3 [3]float32) {
		appendTexturedQuadUV(
			batch, tintR, tintG, tintB,
			v0, 0, 1,
			v1, 0, 0,
			v2, 1, 0,
			v3, 1, 1,
		)
		appendTexturedQuadUV(
			batch, tintR, tintG, tintB,
			v3, 1, 1,
			v2, 1, 0,
			v1, 0, 0,
			v0, 0, 1,
		)
	}
	appendVineDoubleSidedB := func(v0, v1, v2, v3 [3]float32) {
		appendTexturedQuadUV(
			batch, tintR, tintG, tintB,
			v0, 1, 0,
			v1, 1, 1,
			v2, 0, 1,
			v3, 0, 0,
		)
		appendTexturedQuadUV(
			batch, tintR, tintG, tintB,
			v3, 0, 0,
			v2, 0, 1,
			v1, 1, 1,
			v0, 1, 0,
		)
	}
	if (renderMeta & 2) != 0 {
		xp := x0 + inset
		appendVineDoubleSidedA(
			[3]float32{xp, y1, z1},
			[3]float32{xp, y0, z1},
			[3]float32{xp, y0, z0},
			[3]float32{xp, y1, z0},
		)
	}
	if (renderMeta & 8) != 0 {
		xp := x1 - inset
		appendVineDoubleSidedB(
			[3]float32{xp, y0, z1},
			[3]float32{xp, y1, z1},
			[3]float32{xp, y1, z0},
			[3]float32{xp, y0, z0},
		)
	}
	if (renderMeta & 4) != 0 {
		zp := z0 + inset
		appendVineDoubleSidedB(
			[3]float32{x1, y0, zp},
			[3]float32{x1, y1, zp},
			[3]float32{x0, y1, zp},
			[3]float32{x0, y0, zp},
		)
	}
	if (renderMeta & 1) != 0 {
		zp := z1 - inset
		appendVineDoubleSidedA(
			[3]float32{x1, y1, zp},
			[3]float32{x1, y0, zp},
			[3]float32{x0, y0, zp},
			[3]float32{x0, y1, zp},
		)
	}

	if aboveOK && isNormalCubeRenderBlock(aboveID) {
		yp := y1 - inset
		appendTexturedQuadUV(
			batch, tintR, tintG, tintB,
			[3]float32{x1, yp, z0},
			0, 1,
			[3]float32{x1, yp, z1},
			0, 0,
			[3]float32{x0, yp, z1},
			1, 0,
			[3]float32{x0, yp, z0},
			1, 1,
		)
	}

	return true
}

func (a *App) inferSingleVineAttachmentMeta(x, y, z int) int {
	type candidate struct {
		bit int
		nx  int
		nz  int
	}
	// Bit mapping from BlockVine/Direction in 1.6.4:
	// bit2->west, bit8->east, bit4->north, bit1->south support.
	cands := [...]candidate{
		{bit: 2, nx: -1, nz: 0},
		{bit: 8, nx: 1, nz: 0},
		{bit: 4, nx: 0, nz: -1},
		{bit: 1, nx: 0, nz: 1},
	}

	bestBit := 0
	bestScore := -1
	for _, c := range cands {
		id, _, ok := a.session.BlockAt(x+c.nx, y, z+c.nz)
		if !ok || id == 0 {
			continue
		}
		if !isVineSupportRenderBlock(id) {
			continue
		}

		score := 10
		// Prefer obvious tree supports so legacy meta=0 vines stick to trunks/canopy,
		// matching expected jungle/swamp visuals.
		switch id {
		case 17:
			score = 100
		case 18:
			score = 90
		}
		if score > bestScore {
			bestScore = score
			bestBit = c.bit
		}
	}
	return bestBit
}

func isVineSupportRenderBlock(id int) bool {
	if id == 18 {
		return true // leaves are valid vine supports in 1.6.4.
	}
	return isNormalCubeRenderBlock(id)
}

func (a *App) appendFlatPlantBatches(
	x, y, z float32,
	blockX, blockY, blockZ int,
	blockID int,
	solidBatches map[uint32]*textureBatch,
) bool {
	tex := a.blockTextureForFace(blockID, faceUp)
	if tex == nil {
		tex = a.blockTextureForFace(blockID, faceNorth)
	}
	if tex == nil {
		return false
	}

	tintR, tintG, tintB := a.blockFaceTintAt(blockX, blockY, blockZ, blockID, 0, faceUp)
	batch := ensureTextureBatch(solidBatches, tex)
	if batch == nil {
		return false
	}

	// Translation reference:
	// - net.minecraft.src.BlockLilyPad#setBlockBounds(...)
	// Vanilla lily pad is an ultra-thin top quad.
	y0 := y + 0.015625
	appendTexturedQuadUV(
		batch, tintR, tintG, tintB,
		[3]float32{x, y0, z}, 0, 1,
		[3]float32{x, y0, z + 1}, 0, 0,
		[3]float32{x + 1, y0, z + 1}, 1, 0,
		[3]float32{x + 1, y0, z}, 1, 1,
	)
	return true
}

// Translation reference:
// - net.minecraft.src.RenderBlocks#renderCrossedSquares(...)
// - net.minecraft.src.RenderBlocks#drawCrossedSquares(...)
func (a *App) appendCrossedPlantBatches(
	x, y, z float32,
	blockX, blockY, blockZ int,
	blockID int,
	blockMeta int,
	solidBatches map[uint32]*textureBatch,
) bool {
	tex := a.blockTextureForFaceMeta(blockID, blockMeta, faceNorth)
	if tex == nil {
		tex = a.blockTextureForFaceMeta(blockID, blockMeta, faceUp)
	}
	if tex == nil {
		return false
	}

	// Vanilla random wobble for tall grass only.
	if blockID == 31 {
		ix := int64(x)
		iy := int64(y)
		iz := int64(z)
		seed := (ix * 3129871) ^ (iz * 116129781) ^ iy
		seed = seed*seed*42317861 + seed*11
		x += (float32(float64((seed>>16)&15)/15.0) - 0.5) * 0.5
		y += (float32(float64((seed>>20)&15)/15.0) - 1.0) * 0.2
		z += (float32(float64((seed>>24)&15)/15.0) - 0.5) * 0.5
	}

	tintR, tintG, tintB := a.blockFaceTintAt(blockX, blockY, blockZ, blockID, blockMeta, faceNorth)
	batch := ensureTextureBatch(solidBatches, tex)
	if batch == nil {
		return false
	}

	const (
		size   = 0.45
		height = 1.0
	)
	x0 := x + 0.5 - size
	x1 := x + 0.5 + size
	z0 := z + 0.5 - size
	z1 := z + 0.5 + size
	y0 := y
	y1 := y + height

	// Translation reference:
	// - net.minecraft.src.RenderBlocks#drawCrossedSquares(...)
	// Match vanilla's explicit vertex+UV order (4 quads, including backside).
	// Renderer note: block textures are uploaded with vertical flip in loadTexture2D,
	// so V is inverted vs raw MCP Tessellator convention.
	appendTexturedQuadUV(
		batch, tintR, tintG, tintB,
		[3]float32{x0, y1, z0}, 0, 1,
		[3]float32{x0, y0, z0}, 0, 0,
		[3]float32{x1, y0, z1}, 1, 0,
		[3]float32{x1, y1, z1}, 1, 1,
	)
	appendTexturedQuadUV(
		batch, tintR, tintG, tintB,
		[3]float32{x1, y1, z1}, 0, 1,
		[3]float32{x1, y0, z1}, 0, 0,
		[3]float32{x0, y0, z0}, 1, 0,
		[3]float32{x0, y1, z0}, 1, 1,
	)
	appendTexturedQuadUV(
		batch, tintR, tintG, tintB,
		[3]float32{x0, y1, z1}, 0, 1,
		[3]float32{x0, y0, z1}, 0, 0,
		[3]float32{x1, y0, z0}, 1, 0,
		[3]float32{x1, y1, z0}, 1, 1,
	)
	appendTexturedQuadUV(
		batch, tintR, tintG, tintB,
		[3]float32{x1, y1, z0}, 0, 1,
		[3]float32{x1, y0, z0}, 0, 0,
		[3]float32{x0, y0, z1}, 1, 0,
		[3]float32{x0, y1, z1}, 1, 1,
	)
	return true
}

func (a *App) appendFallbackBlockBatches(
	x, y, z, r, g, b float32,
	faces visibleFaces,
	solidBatches map[uint32]*textureBatch,
) {
	batch := ensureTextureBatch(solidBatches, nil)
	if batch == nil {
		return
	}
	x2, y2, z2 := x+1, y+1, z+1

	bright := func(v float32) float32 {
		return float32(math.Min(float64(v+0.18), 1.0))
	}
	dim := func(v float32) float32 {
		return float32(math.Max(float64(v-0.18), 0.0))
	}

	appendFace := func(rr, gg, bb float32, v0, v1, v2, v3 [3]float32) {
		appendTexturedQuad(batch, rr, gg, bb, v0, v1, v2, v3)
	}
	if faces.Up {
		appendFace(bright(r), bright(g), bright(b),
			[3]float32{x, y2, z},
			[3]float32{x, y2, z2},
			[3]float32{x2, y2, z2},
			[3]float32{x2, y2, z},
		)
	}
	if faces.Down {
		appendFace(dim(r), dim(g), dim(b),
			[3]float32{x, y, z},
			[3]float32{x2, y, z},
			[3]float32{x2, y, z2},
			[3]float32{x, y, z2},
		)
	}
	if faces.North {
		appendFace(r, g, b,
			[3]float32{x2, y, z},
			[3]float32{x, y, z},
			[3]float32{x, y2, z},
			[3]float32{x2, y2, z},
		)
	}
	if faces.South {
		appendFace(r, g, b,
			[3]float32{x, y, z2},
			[3]float32{x2, y, z2},
			[3]float32{x2, y2, z2},
			[3]float32{x, y2, z2},
		)
	}
	if faces.West {
		appendFace(dim(r), dim(g), dim(b),
			[3]float32{x, y, z},
			[3]float32{x, y, z2},
			[3]float32{x, y2, z2},
			[3]float32{x, y2, z},
		)
	}
	if faces.East {
		appendFace(r, g, b,
			[3]float32{x2, y, z2},
			[3]float32{x2, y, z},
			[3]float32{x2, y2, z},
			[3]float32{x2, y2, z2},
		)
	}
}

func ensureTextureBatch(batches map[uint32]*textureBatch, tex *texture2D) *textureBatch {
	texID := uint32(0)
	if tex != nil {
		texID = tex.ID
	}
	b := batches[texID]
	if b != nil {
		return b
	}
	b = &textureBatch{tex: tex}
	batches[texID] = b
	return b
}

func appendTexturedQuad(batch *textureBatch, r, g, b float32, v0, v1, v2, v3 [3]float32) {
	appendTexturedQuadUV(
		batch,
		r, g, b,
		v0, 0, 0,
		v1, 1, 0,
		v2, 1, 1,
		v3, 0, 1,
	)
}

func appendTexturedQuadUV(
	batch *textureBatch,
	r, g, b float32,
	p0 [3]float32, u0, v0 float32,
	p1 [3]float32, u1, v1 float32,
	p2 [3]float32, u2, v2 float32,
	p3 [3]float32, u3, v3 float32,
) {
	if batch == nil {
		return
	}
	batch.verts = append(batch.verts,
		texturedVertex{x: p0[0], y: p0[1], z: p0[2], u: u0, v: v0, r: r, g: g, b: b},
		texturedVertex{x: p1[0], y: p1[1], z: p1[2], u: u1, v: v1, r: r, g: g, b: b},
		texturedVertex{x: p2[0], y: p2[1], z: p2[2], u: u2, v: v2, r: r, g: g, b: b},
		texturedVertex{x: p3[0], y: p3[1], z: p3[2], u: u3, v: v3, r: r, g: g, b: b},
	)
}

func (a *App) emitTextureBatches(batches map[uint32]*textureBatch) {
	if len(batches) == 0 {
		return
	}

	ids := make([]int, 0, len(batches))
	for id, batch := range batches {
		if batch == nil || len(batch.verts) == 0 {
			continue
		}
		ids = append(ids, int(id))
	}
	if len(ids) == 0 {
		return
	}
	sort.Ints(ids)

	for _, id := range ids {
		batch := batches[uint32(id)]
		if batch == nil || len(batch.verts) == 0 {
			continue
		}

		if batch.tex != nil {
			gl.Enable(gl.TEXTURE_2D)
			batch.tex.bind()
		} else {
			gl.Disable(gl.TEXTURE_2D)
		}

		gl.Begin(gl.QUADS)
		for _, v := range batch.verts {
			gl.Color3f(v.r, v.g, v.b)
			if batch.tex != nil {
				gl.TexCoord2f(v.u, v.v)
			}
			gl.Vertex3f(v.x, v.y, v.z)
		}
		gl.End()
	}

	gl.Enable(gl.TEXTURE_2D)
	gl.Color4f(1, 1, 1, 1)
}

func (a *App) drawEntities() {
	entities := a.session.EntitiesSnapshot()
	if len(entities) == 0 {
		return
	}

	// Temporary safety: model winding parity is still being aligned to vanilla;
	// disabling cull avoids visual "holes"/transparent-looking limbs.
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.ALPHA_TEST)
	gl.AlphaFunc(gl.GREATER, 0.1)
	gl.Disable(gl.TEXTURE_2D)
	gl.Enable(gl.DEPTH_TEST)
	animTime := float64(time.Now().UnixNano()) / 1e9
	for _, ent := range entities {
		a.drawEntityModel(ent, animTime)
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.CULL_FACE)
	gl.Color4f(1, 1, 1, 1)
}

func (a *App) startHandSwing() {
	a.handSwingStart = time.Now()
}

func (a *App) currentHandSwingProgress(now time.Time) float32 {
	const swingDuration = 300 * time.Millisecond // 6 ticks at 20 TPS
	if a.handSwingStart.IsZero() {
		return 0
	}
	elapsed := now.Sub(a.handSwingStart)
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= swingDuration {
		a.handSwingStart = time.Time{}
		return 0
	}
	return float32(float64(elapsed) / float64(swingDuration))
}

// Translation references:
// - net.minecraft.src.ItemRenderer#renderItemInFirstPerson(float)
// - net.minecraft.src.RenderPlayer#renderFirstPersonArm(EntityPlayer)
func (a *App) drawFirstPersonArm(_ netclient.StateSnapshot) {
	if a.mainMenu || a.session == nil || a.hudHidden {
		return
	}

	tex := a.entityTextureForType(0)
	aspect := float64(a.width) / float64(maxInt(a.height, 1))

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	setPerspective(a.currentFOVDegrees(), aspect, 0.05, 10.0)

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	gl.LoadIdentity()

	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.ALPHA_TEST)
	gl.AlphaFunc(gl.GREATER, 0.1)
	gl.Enable(gl.TEXTURE_2D)

	partial := float32(1.0)
	if moveTickSeconds > 0 {
		partial = float32(a.renderLerpTime / moveTickSeconds)
		if partial < 0 {
			partial = 0
		} else if partial > 1 {
			partial = 1
		}
	}
	armPitch := float32(a.prevRenderArmPitch + (a.renderArmPitch-a.prevRenderArmPitch)*float64(partial))
	armYaw := float32(a.prevRenderArmYaw + (a.renderArmYaw-a.prevRenderArmYaw)*float64(partial))
	gl.Rotatef((float32(a.pitch)-armPitch)*0.1, 1, 0, 0)
	gl.Rotatef((float32(a.yaw)-armYaw)*0.1, 0, 1, 0)

	// Translation reference:
	// - net.minecraft.src.ItemRenderer#renderItemInFirstPerson(float)
	//   empty-hand branch (else if !player.isInvisible()).
	swing := a.currentHandSwingProgress(time.Now())
	sinSwing := float32(math.Sin(float64(swing) * math.Pi))
	sinSqrtSwing := float32(math.Sin(math.Sqrt(float64(swing)) * math.Pi))
	equip := float32(1.0)
	armScale := float32(0.8)

	gl.Translatef(-sinSqrtSwing*0.3, float32(math.Sin(math.Sqrt(float64(swing))*math.Pi*2.0))*0.4, -sinSwing*0.4)
	gl.Translatef(0.8*armScale, -0.75*armScale-(1.0-equip)*0.6, -0.9*armScale)
	gl.Rotatef(45.0, 0, 1, 0)

	sinSwing2 := float32(math.Sin(float64(swing*swing) * math.Pi))
	sinSqrtSwing2 := float32(math.Sin(math.Sqrt(float64(swing)) * math.Pi))
	gl.Rotatef(sinSqrtSwing2*70.0, 0, 1, 0)
	gl.Rotatef(-sinSwing2*20.0, 0, 0, 1)

	gl.Translatef(-1.0, 3.6, 3.5)
	gl.Rotatef(120.0, 0, 0, 1)
	gl.Rotatef(200.0, 1, 0, 0)
	gl.Rotatef(-135.0, 0, 1, 0)
	gl.Translatef(5.6, 0.0, 0.0)

	a.drawFirstPersonRightArmRaw(tex)

	gl.Enable(gl.CULL_FACE)
	gl.Color4f(1, 1, 1, 1)

	gl.PopMatrix()
	gl.MatrixMode(gl.PROJECTION)
	gl.PopMatrix()
	gl.MatrixMode(gl.MODELVIEW)
}

func (a *App) drawFirstPersonRightArmRaw(tex *texture2D) {
	// Translation reference:
	// - net.minecraft.src.RenderPlayer#renderFirstPersonArm(EntityPlayer)
	// - net.minecraft.src.ModelBiped constructor for bipedRightArm
	// RenderPlayer uses ModelRenderer raw coordinates directly (without 24-y remap used by
	// this rewrite's world-space entity renderer path), so first-person arm needs dedicated geometry.
	armUV := cuboidUVFromTextureOffset(tex, 40, 16, 4, 12, 4)
	const inv16 = float32(1.0 / 16.0)

	gl.PushMatrix()
	gl.Translatef(-5.0*inv16, 2.0*inv16, 0.0)
	drawModelPartUVRawEx(
		-1.0*inv16,
		4.0*inv16,
		0.0,
		4.0*inv16,
		12.0*inv16,
		4.0*inv16,
		tex,
		armUV,
		false,
		1,
		1,
		1,
	)
	gl.PopMatrix()
}

func (a *App) drawTexturedBlock(x, y, z float32, blockID int, faces visibleFaces) bool {
	if len(a.blockTextureDefs) == 0 || len(a.blockTextures) == 0 {
		return false
	}
	if _, ok := a.blockTextureDefs[blockID]; !ok {
		return false
	}

	top := a.blockTextureForFace(blockID, faceUp)
	bottom := a.blockTextureForFace(blockID, faceDown)
	north := a.blockTextureForFace(blockID, faceNorth)
	south := a.blockTextureForFace(blockID, faceSouth)
	west := a.blockTextureForFace(blockID, faceWest)
	east := a.blockTextureForFace(blockID, faceEast)

	if (faces.Up && top == nil) ||
		(faces.Down && bottom == nil) ||
		(faces.North && north == nil) ||
		(faces.South && south == nil) ||
		(faces.West && west == nil) ||
		(faces.East && east == nil) {
		return false
	}

	x2, y2, z2 := x+1, y+1, z+1
	drawFaceWithTint := func(tex *texture2D, shade, tintR, tintG, tintB float32, v0, v1, v2, v3 [3]float32) {
		if tex == nil {
			return
		}
		tex.bind()
		gl.Color4f(tintR*shade, tintG*shade, tintB*shade, 1)
		gl.Begin(gl.QUADS)
		gl.TexCoord2f(0, 0)
		gl.Vertex3f(v0[0], v0[1], v0[2])
		gl.TexCoord2f(1, 0)
		gl.Vertex3f(v1[0], v1[1], v1[2])
		gl.TexCoord2f(1, 1)
		gl.Vertex3f(v2[0], v2[1], v2[2])
		gl.TexCoord2f(0, 1)
		gl.Vertex3f(v3[0], v3[1], v3[2])
		gl.End()
	}
	drawFace := func(tex *texture2D, face int, shade float32, v0, v1, v2, v3 [3]float32) {
		tintR, tintG, tintB := a.blockFaceTint(blockID, face)
		drawFaceWithTint(tex, shade, tintR, tintG, tintB, v0, v1, v2, v3)
	}

	gl.Enable(gl.TEXTURE_2D)
	if faces.Up {
		drawFace(top, faceUp, 1.0,
			[3]float32{x, y2, z},
			[3]float32{x, y2, z2},
			[3]float32{x2, y2, z2},
			[3]float32{x2, y2, z},
		)
	}
	if faces.Down {
		drawFace(bottom, faceDown, 0.52,
			[3]float32{x, y, z},
			[3]float32{x2, y, z},
			[3]float32{x2, y, z2},
			[3]float32{x, y, z2},
		)
	}
	if faces.North {
		drawFace(north, faceNorth, 0.80,
			[3]float32{x2, y, z},
			[3]float32{x, y, z},
			[3]float32{x, y2, z},
			[3]float32{x2, y2, z},
		)
	}
	if faces.South {
		drawFace(south, faceSouth, 0.80,
			[3]float32{x, y, z2},
			[3]float32{x2, y, z2},
			[3]float32{x2, y2, z2},
			[3]float32{x, y2, z2},
		)
	}
	if faces.West {
		drawFace(west, faceWest, 0.65,
			[3]float32{x, y, z},
			[3]float32{x, y, z2},
			[3]float32{x, y2, z2},
			[3]float32{x, y2, z},
		)
	}
	if faces.East {
		drawFace(east, faceEast, 0.65,
			[3]float32{x2, y, z2},
			[3]float32{x2, y, z},
			[3]float32{x2, y2, z},
			[3]float32{x2, y2, z2},
		)
	}

	overlay := a.blockSideOverlayTexture(blockID)
	if overlay != nil {
		overlayR, overlayG, overlayB := a.blockSideOverlayTint(blockID)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		if faces.North {
			drawFaceWithTint(overlay, 0.80, overlayR, overlayG, overlayB,
				[3]float32{x2, y, z},
				[3]float32{x, y, z},
				[3]float32{x, y2, z},
				[3]float32{x2, y2, z},
			)
		}
		if faces.South {
			drawFaceWithTint(overlay, 0.80, overlayR, overlayG, overlayB,
				[3]float32{x, y, z2},
				[3]float32{x2, y, z2},
				[3]float32{x2, y2, z2},
				[3]float32{x, y2, z2},
			)
		}
		if faces.West {
			drawFaceWithTint(overlay, 0.65, overlayR, overlayG, overlayB,
				[3]float32{x, y, z},
				[3]float32{x, y, z2},
				[3]float32{x, y2, z2},
				[3]float32{x, y2, z},
			)
		}
		if faces.East {
			drawFaceWithTint(overlay, 0.65, overlayR, overlayG, overlayB,
				[3]float32{x2, y, z2},
				[3]float32{x2, y, z},
				[3]float32{x2, y2, z},
				[3]float32{x2, y2, z2},
			)
		}
		gl.Disable(gl.BLEND)
	}

	gl.Color4f(1, 1, 1, 1)
	return true
}

func (a *App) drawHUD(snap netclient.StateSnapshot) {
	if a.hudHidden {
		return
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(uiW, uiH)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	if !a.inventoryOpen {
		if a.texWidgets != nil {
			gl.Enable(gl.TEXTURE_2D)
			gl.Color4f(1, 1, 1, 1)
			drawTexturedRect(a.texWidgets, float32(uiW/2-91), float32(uiH-22), 182, 22, 0, 0, 182, 22)
			held := int(snap.HeldSlot)
			if held < 0 {
				held = 0
			}
			if held > 8 {
				held = 8
			}
			drawTexturedRect(a.texWidgets, float32(uiW/2-91-1+held*20), float32(uiH-23), 24, 22, 0, 22, 24, 22)
		}

		if a.texIcons != nil {
			gl.Enable(gl.TEXTURE_2D)
			gl.Color4f(1, 1, 1, 1)
			gl.BlendFunc(gl.ONE_MINUS_DST_COLOR, gl.ONE_MINUS_SRC_COLOR)
			drawTexturedRect(a.texIcons, float32(uiW/2-7), float32(uiH/2-7), 16, 16, 0, 0, 16, 16)
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		}

		a.drawHealthAndFood(snap)
		a.drawExperienceBar(snap)
		a.drawHotbarStackCounts(snap)
		a.drawDebugOverlay(snap)
		a.drawPlayerListOverlay(snap)
	}
	a.drawChatOverlay()
	a.drawChatInput()

	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

// Translation reference:
// - net.minecraft.src.GuiIngame.func_110327_a()
func (a *App) drawHealthAndFood(snap netclient.StateSnapshot) {
	if a.texIcons == nil {
		return
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()

	gl.Enable(gl.TEXTURE_2D)
	gl.Color4f(1, 1, 1, 1)
	left := uiW/2 - 91
	right := uiW/2 + 91
	y := uiH - 39

	health := int(math.Ceil(float64(snap.Health)))
	if health < 0 {
		health = 0
	}
	if health > 20 {
		health = 20
	}

	for i := 0; i < 10; i++ {
		x := left + i*8
		drawTexturedRect(a.texIcons, float32(x), float32(y), 9, 9, 16, 0, 9, 9)
		if i*2+1 < health {
			drawTexturedRect(a.texIcons, float32(x), float32(y), 9, 9, 52, 0, 9, 9)
		} else if i*2+1 == health {
			drawTexturedRect(a.texIcons, float32(x), float32(y), 9, 9, 61, 0, 9, 9)
		}
	}

	food := int(snap.Food)
	if food < 0 {
		food = 0
	}
	if food > 20 {
		food = 20
	}
	for i := 0; i < 10; i++ {
		x := right - i*8 - 9
		drawTexturedRect(a.texIcons, float32(x), float32(y), 9, 9, 16, 27, 9, 9)
		if i*2+1 < food {
			drawTexturedRect(a.texIcons, float32(x), float32(y), 9, 9, 52, 27, 9, 9)
		} else if i*2+1 == food {
			drawTexturedRect(a.texIcons, float32(x), float32(y), 9, 9, 61, 27, 9, 9)
		}
	}
}

func (a *App) drawHotbarStackCounts(snap netclient.StateSnapshot) {
	if a.font == nil {
		return
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()

	baseX := uiW/2 - 90
	baseY := uiH - 16 - 3
	for i := 0; i < 9; i++ {
		slot := snap.Hotbar[i]
		if slot.ItemID == 0 || slot.StackSize <= 1 {
			continue
		}
		count := strconv.Itoa(int(slot.StackSize))
		x := baseX + i*20 + 2 + 19 - a.font.getStringWidth(count)
		y := baseY + 9
		a.font.drawStringWithShadow(count, x, y, 0xFFFFFF)
	}
}

// Translation reference:
// - net.minecraft.src.GuiIngame.renderGameOverlay() exp bar + level
func (a *App) drawExperienceBar(snap netclient.StateSnapshot) {
	if a.texIcons == nil {
		return
	}
	// Translation reference:
	// - net.minecraft.src.GuiIngame.renderGameOverlay()
	// In vanilla 1.6.4 survival/adventure, the XP bar background is rendered
	// whenever xpBarCap() > 0 (including level 0 with 0 progress).
	// Creative mode suppresses this bar.
	if snap.IsCreative {
		return
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()

	baseX := uiW/2 - 91
	y := uiH - 32 + 3
	gl.Enable(gl.TEXTURE_2D)
	gl.Color4f(1, 1, 1, 1)
	drawTexturedRect(a.texIcons, float32(baseX), float32(y), 182, 5, 0, 64, 182, 5)

	fill := int(snap.ExperienceBar * 183.0)
	if fill < 0 {
		fill = 0
	}
	if fill > 182 {
		fill = 182
	}
	if fill > 0 {
		drawTexturedRect(a.texIcons, float32(baseX), float32(y), float32(fill), 5, 0, 69, fill, 5)
	}

	if snap.ExperienceLvl > 0 && a.font != nil {
		level := strconv.Itoa(int(snap.ExperienceLvl))
		cx := uiW/2 - a.font.getStringWidth(level)/2
		ty := uiH - 31 - 4
		a.font.drawString(level, cx+1, ty, 0x000000)
		a.font.drawString(level, cx-1, ty, 0x000000)
		a.font.drawString(level, cx, ty+1, 0x000000)
		a.font.drawString(level, cx, ty-1, 0x000000)
		a.font.drawString(level, cx, ty, 0x80FF20)
	}
}

func (a *App) drawDebugOverlay(snap netclient.StateSnapshot) {
	if !a.showDebug || a.font == nil {
		return
	}

	facing := facingFromYaw(float64(snap.PlayerYaw))
	lines := []string{
		"Minecraft 1.6.4 (GoMC dev)",
		fmt.Sprintf("%d fps", a.currentFPS),
		fmt.Sprintf("XYZ: %.3f / %.3f / %.3f", snap.PlayerX, snap.PlayerY, snap.PlayerZ),
		fmt.Sprintf("Yaw/Pitch: %.2f / %.2f", snap.PlayerYaw, snap.PlayerPitch),
		fmt.Sprintf("Facing: %s", facing),
		fmt.Sprintf("Chunks: %d  Entities: %d", snap.LoadedChunks, snap.TrackedEntities),
		fmt.Sprintf("RenderDist: %s (%d chunks)  Cache: %d", a.renderDistanceModeName(), a.renderDistanceChunks(), len(a.chunkRenderCache)),
		fmt.Sprintf("Health/Food: %.1f / %d", snap.Health, snap.Food),
		fmt.Sprintf("Time: %d / %d", snap.WorldAge, snap.WorldTime),
	}
	y := 2
	for _, line := range lines {
		a.font.drawStringWithShadow(line, 2, y, 0xFFFFFF)
		y += 10
	}
}

func (a *App) drawPlayerListOverlay(snap netclient.StateSnapshot) {
	if !a.showPlayerList || a.session == nil || a.font == nil {
		return
	}

	players := a.session.PlayerListSnapshot()
	if len(players) == 0 && strings.TrimSpace(snap.Username) != "" {
		players = append(players, netclient.PlayerInfoSnapshot{Name: snap.Username, Ping: 0})
	}
	if len(players) == 0 {
		return
	}

	uiW := a.uiWidth()
	title := fmt.Sprintf("Players online: %d", len(players))
	maxW := a.font.getStringWidth(title)
	names := make([]string, 0, len(players))
	for i := range players {
		name := strings.TrimSpace(players[i].Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		if w := a.font.getStringWidth(name); w > maxW {
			maxW = w
		}
	}
	if len(names) == 0 {
		return
	}

	panelW := maxW + 16
	panelH := 20 + len(names)*10 + 6
	x1 := uiW/2 - panelW/2
	y1 := 18
	x2 := x1 + panelW
	y2 := y1 + panelH

	drawSolidRect(x1, y1, x2, y2, 0x90000000)
	drawSolidRect(x1, y1, x2, y1+1, 0x50FFFFFF)
	drawSolidRect(x1, y2-1, x2, y2, 0x70000000)
	a.font.drawCenteredString(title, uiW/2, y1+6, 0xFFFFFF)

	y := y1 + 20
	for _, name := range names {
		a.font.drawCenteredString(name, uiW/2, y, 0xFFFFFF)
		y += 10
	}
}

func (a *App) drawChatOverlay() {
	if a.font == nil {
		return
	}
	uiH := a.uiHeight()

	a.chatMu.Lock()
	lines := make([]chatLine, len(a.chatLines))
	copy(lines, a.chatLines)
	a.chatMu.Unlock()

	if len(lines) == 0 {
		return
	}

	maxLines := 10
	if a.chatInputOpen {
		maxLines = 20
	}
	now := time.Now()
	baseY := uiH - 48
	if a.chatInputOpen {
		baseY = uiH - 28
	}
	drawn := 0

	for i := len(lines) - 1; i >= 0 && drawn < maxLines; i-- {
		line := lines[i]
		age := now.Sub(line.AddedAt)
		if !a.paused && !a.chatInputOpen && age > 10*time.Second {
			continue
		}

		alpha := 255
		if !a.paused && !a.chatInputOpen {
			fade := int((10*time.Second - age) * 255 / (10 * time.Second))
			if fade < 0 {
				fade = 0
			}
			if fade > 255 {
				fade = 255
			}
			alpha = fade
		}
		if alpha <= 8 {
			continue
		}

		text := line.Message
		w := a.font.getStringWidth(text)
		y := baseY - drawn*10
		bg := (alpha / 2) << 24
		drawSolidRect(1, y-1, 2+w+1, y+9, bg)
		color := (alpha << 24) | (line.Color & 0xFFFFFF)
		a.font.drawStringWithShadow(text, 2, y, color)
		drawn++
	}
}

func (a *App) drawChatInput() {
	if !a.chatInputOpen || a.font == nil {
		return
	}
	uiW, uiH := a.uiWidth(), a.uiHeight()
	drawSolidRect(2, uiH-14, uiW-2, uiH-2, 0x80000000)
	cursor := ""
	if (time.Now().UnixMilli()/500)%2 == 0 {
		cursor = "_"
	}
	a.font.drawStringWithShadow(a.chatInput+cursor, 4, uiH-12, 0xE0E0E0)
}

func (a *App) hasMenuPanorama() bool {
	if a.texMenuView == nil {
		return false
	}
	for i := range a.texPanorama {
		if a.texPanorama[i] == nil {
			return false
		}
	}
	return true
}

// Translation reference:
// - net.minecraft.src.GuiMainMenu.drawPanorama()
func (a *App) drawPanoramaCube(partial float32) {
	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	setPerspective(120.0, 1.0, 0.05, 10.0)

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Color4f(1, 1, 1, 1)
	gl.Rotatef(180, 1, 0, 0)
	gl.Enable(gl.BLEND)
	gl.Disable(gl.ALPHA_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.DepthMask(false)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	const tiles = 8
	rotTick := float32(a.panoramaTick) + partial
	for tile := 0; tile < tiles*tiles; tile++ {
		gl.PushMatrix()
		tx := (float32(tile%tiles)/float32(tiles) - 0.5) / 64.0
		ty := (float32(tile/tiles)/float32(tiles) - 0.5) / 64.0
		gl.Translatef(tx, ty, 0)
		gl.Rotatef(float32(math.Sin(float64(rotTick)/400.0))*25.0+20.0, 1, 0, 0)
		gl.Rotatef(-rotTick*0.1, 0, 1, 0)

		alpha := 1.0 / float32(tile+1)
		for face := 0; face < 6; face++ {
			gl.PushMatrix()
			switch face {
			case 1:
				gl.Rotatef(90, 0, 1, 0)
			case 2:
				gl.Rotatef(180, 0, 1, 0)
			case 3:
				gl.Rotatef(-90, 0, 1, 0)
			case 4:
				gl.Rotatef(90, 1, 0, 0)
			case 5:
				gl.Rotatef(-90, 1, 0, 0)
			}

			gl.Enable(gl.TEXTURE_2D)
			a.texPanorama[face].bind()
			gl.Color4f(1, 1, 1, alpha)
			gl.Begin(gl.QUADS)
			gl.TexCoord2f(0, 0)
			gl.Vertex3f(-1, -1, 1)
			gl.TexCoord2f(1, 0)
			gl.Vertex3f(1, -1, 1)
			gl.TexCoord2f(1, 1)
			gl.Vertex3f(1, 1, 1)
			gl.TexCoord2f(0, 1)
			gl.Vertex3f(-1, 1, 1)
			gl.End()
			gl.PopMatrix()
		}

		gl.PopMatrix()
		gl.ColorMask(true, true, true, false)
	}

	gl.ColorMask(true, true, true, true)
	gl.MatrixMode(gl.PROJECTION)
	gl.PopMatrix()
	gl.MatrixMode(gl.MODELVIEW)
	gl.PopMatrix()
	gl.DepthMask(true)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.ALPHA_TEST)
	gl.Enable(gl.DEPTH_TEST)
	gl.Color4f(1, 1, 1, 1)
}

// Translation reference:
// - net.minecraft.src.GuiMainMenu.rotateAndBlurSkybox()
func (a *App) rotateAndBlurSkybox() {
	if a.texMenuView == nil {
		return
	}

	a.texMenuView.bind()
	gl.CopyTexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, 0, 0, 256, 256)
	gl.Enable(gl.BLEND)
	gl.Enable(gl.TEXTURE_2D)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.ColorMask(true, true, true, false)
	gl.Disable(gl.ALPHA_TEST)
	gl.Disable(gl.DEPTH_TEST)
	begin2D(a.width, a.height)

	for pass := 0; pass < 3; pass++ {
		offset := float32(pass-1) / 256.0
		alpha := 1.0 / float32(pass+1)
		gl.Color4f(1, 1, 1, alpha)
		gl.Begin(gl.QUADS)
		gl.TexCoord2f(0.0+offset, 0.0)
		gl.Vertex2f(float32(a.width), float32(a.height))
		gl.TexCoord2f(1.0+offset, 0.0)
		gl.Vertex2f(float32(a.width), 0)
		gl.TexCoord2f(1.0+offset, 1.0)
		gl.Vertex2f(0, 0)
		gl.TexCoord2f(0.0+offset, 1.0)
		gl.Vertex2f(0, float32(a.height))
		gl.End()
	}

	gl.ColorMask(true, true, true, true)
	gl.Enable(gl.ALPHA_TEST)
	gl.Color4f(1, 1, 1, 1)
}

// Translation reference:
// - net.minecraft.src.GuiMainMenu.renderSkybox()
func (a *App) renderSkybox(partial float32) {
	if !a.hasMenuPanorama() {
		return
	}

	gl.Viewport(0, 0, 256, 256)
	a.drawPanoramaCube(partial)
	gl.Enable(gl.TEXTURE_2D)
	for i := 0; i < 8; i++ {
		a.rotateAndBlurSkybox()
	}

	gl.Viewport(0, 0, int32(a.width), int32(a.height))
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(a.width, a.height)
	gl.Enable(gl.TEXTURE_2D)
	a.texMenuView.bind()
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int32(gl.LINEAR))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int32(gl.LINEAR))
	gl.Color4f(1, 1, 1, 1)

	scale := float32(120.0 / float32(maxInt(a.width, a.height)))
	uRadius := float32(a.height) * scale / 256.0
	vRadius := float32(a.width) * scale / 256.0

	gl.Begin(gl.QUADS)
	gl.TexCoord2f(0.5-uRadius, 0.5+vRadius)
	gl.Vertex2f(0, float32(a.height))
	gl.TexCoord2f(0.5-uRadius, 0.5-vRadius)
	gl.Vertex2f(float32(a.width), float32(a.height))
	gl.TexCoord2f(0.5+uRadius, 0.5-vRadius)
	gl.Vertex2f(float32(a.width), 0)
	gl.TexCoord2f(0.5+uRadius, 0.5+vRadius)
	gl.Vertex2f(0, 0)
	gl.End()
}

func (a *App) drawMainMenu() {
	uiW, uiH := a.uiWidth(), a.uiHeight()
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	if a.hasMenuPanorama() {
		a.renderSkybox(float32(a.panoramaFrac))
	} else if a.texOptionsBG != nil {
		gl.Disable(gl.DEPTH_TEST)
		gl.Disable(gl.CULL_FACE)
		begin2D(uiW, uiH)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		a.drawTiledOptionsBackground(0, 0, uiW, uiH, 0x404040)
	}

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(uiW, uiH)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// Match vanilla two-layer main menu darkening.
	drawGradientRect(0, 0, uiW, uiH, 0x80FFFFFF, 0x00FFFFFF)
	drawGradientRect(0, 0, uiW, uiH, 0x00000000, 0x80000000)

	if a.texTitle != nil {
		gl.Enable(gl.TEXTURE_2D)
		gl.Color4f(1, 1, 1, 1)
		x := uiW/2 - 155
		y := 30
		drawTexturedRect(a.texTitle, float32(x), float32(y), 155, 44, 0, 0, 155, 44)
		drawTexturedRect(a.texTitle, float32(x+155), float32(y), 155, 44, 0, 45, 155, 44)
	}

	if a.font != nil {
		if a.splashText != "" {
			scale := 1.8 - math.Abs(math.Sin(float64(time.Now().UnixMilli()%1000)/1000.0*math.Pi*2.0))*0.1
			sw := float64(a.font.getStringWidth(a.splashText) + 32)
			if sw > 0 {
				scale = scale * 100.0 / sw
			}
			cx := float32(uiW/2 + 90)
			cy := float32(70)
			gl.PushMatrix()
			gl.Translatef(cx, cy, 0)
			gl.Rotatef(-20.0, 0, 0, 1)
			gl.Scalef(float32(scale), float32(scale), float32(scale))
			a.font.drawCenteredString(a.splashText, 0, -8, 0xFFFF00)
			gl.PopMatrix()
		}

		a.font.drawString("Minecraft 1.6.4", 2, uiH-10, 0xFFFFFF)
		cr := "Copyright Mojang AB. Do not distribute!"
		a.font.drawString(cr, uiW-a.font.getStringWidth(cr)-2, uiH-10, 0xFFFFFF)
	}

	for _, b := range a.mainButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}

	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawPauseMenu() {
	uiW, uiH := a.uiWidth(), a.uiHeight()
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(uiW, uiH)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	if a.texOptionsBG != nil {
		a.drawTiledOptionsBackground(0, 0, uiW, uiH, 0x404040)
	}

	drawGradientRect(0, 0, uiW, uiH, 0xC0101010, 0xD0101010)

	if a.font != nil {
		title := "Game menu"
		if a.pauseScreen == pauseScreenOptions {
			title = "Options"
		} else if a.pauseScreen == pauseScreenVideo {
			title = "Video Settings"
		} else if a.pauseScreen == pauseScreenControls {
			title = "Controls"
		} else if a.pauseScreen == pauseScreenKeyBindings {
			title = "Controls"
		} else if a.pauseScreen == pauseScreenSounds {
			title = "Music & Sounds"
		}
		a.font.drawCenteredString(title, uiW/2, 40, 0xFFFFFF)
	}

	for _, b := range a.currentPauseButtons() {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}
	if a.pauseScreen == pauseScreenVideo && a.font != nil {
		baseY := uiH/6 + 20
		rd := a.optionRenderDistanceLabel()
		fov := a.optionFOVLabel()
		a.font.drawCenteredString(rd, uiW/2, baseY+24+6, 0xFFFFFF)
		a.font.drawCenteredString(fov, uiW/2, baseY+48+6, 0xFFFFFF)
	} else if a.pauseScreen == pauseScreenControls && a.font != nil {
		baseY := uiH/6 + 20
		sens := fmt.Sprintf("Sensitivity: %d%%", a.sensitivityPercent())
		a.font.drawCenteredString(sens, uiW/2, baseY+24+6, 0xFFFFFF)
	} else if a.pauseScreen == pauseScreenSounds && a.font != nil {
		baseY := uiH/6 + 20
		a.font.drawCenteredString(a.optionMusicVolumeLabel(), uiW/2, baseY+6, 0xFFFFFF)
		a.font.drawCenteredString(a.optionSoundVolumeLabel(), uiW/2, baseY+24+6, 0xFFFFFF)
	} else if a.pauseScreen == pauseScreenKeyBindings && a.font != nil {
		a.drawKeyBindingLabels()
	}
	a.drawMenuStatusLine()

	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) resolvePlayerMovement(px, py, pz, dx, dy, dz float64) (float64, float64, float64, bool) {
	// Translation reference:
	// - net.minecraft.src.Entity.moveEntity(double,double,double)
	playerBB := playerBoundingBox(px, py, pz)
	expanded := playerBB.addCoord(dx, dy, dz)
	collisions := a.collisionBoxesForAABB(expanded)

	origDY := dy
	for _, bb := range collisions {
		dy = bb.calculateYOffset(playerBB, dy)
	}
	playerBB = playerBB.offset(0, dy, 0)

	for _, bb := range collisions {
		dx = bb.calculateXOffset(playerBB, dx)
	}
	playerBB = playerBB.offset(dx, 0, 0)

	for _, bb := range collisions {
		dz = bb.calculateZOffset(playerBB, dz)
	}
	playerBB = playerBB.offset(0, 0, dz)

	grounded := origDY < 0 && origDY != dy
	return dx, dy, dz, grounded
}

func (a *App) playerGroundedAt(px, py, pz float64) bool {
	playerBB := playerBoundingBox(px, py, pz)
	query := playerBB.addCoord(0, -0.001, 0)
	collisions := a.collisionBoxesForAABB(query)
	dy := -0.001
	for _, bb := range collisions {
		dy = bb.calculateYOffset(playerBB, dy)
	}
	return dy != -0.001
}

func (a *App) playerCollidesAt(px, py, pz float64) bool {
	playerBB := playerBoundingBox(px, py, pz)
	collisions := a.collisionBoxesForAABB(playerBB)
	for _, bb := range collisions {
		if bb.intersects(playerBB) {
			return true
		}
	}
	return false
}

func (a *App) collisionBoxesForAABB(query axisAlignedBB) []axisAlignedBB {
	if a.session == nil {
		return nil
	}

	const eps = 1e-6
	startX := int(math.Floor(query.minX))
	endX := int(math.Floor(query.maxX - eps))
	startY := int(math.Floor(query.minY))
	endY := int(math.Floor(query.maxY - eps))
	startZ := int(math.Floor(query.minZ))
	endZ := int(math.Floor(query.maxZ - eps))

	boxes := make([]axisAlignedBB, 0, 24)
	for bx := startX; bx <= endX; bx++ {
		for by := startY; by <= endY; by++ {
			for bz := startZ; bz <= endZ; bz++ {
				if by >= 256 {
					boxes = append(boxes, unitBlockAABB(bx, by, bz))
					continue
				}
				if by < 0 {
					continue
				}

				id, meta, ok := a.session.BlockAt(bx, by, bz)
				if !ok {
					// Keep movement conservative when chunk data is unavailable.
					boxes = append(boxes, unitBlockAABB(bx, by, bz))
					continue
				}

				collision := blockCollisionRelativeAABBs(id, meta)
				for _, bb := range collision {
					boxes = append(boxes, axisAlignedBB{
						minX: float64(bx) + bb.minX,
						minY: float64(by) + bb.minY,
						minZ: float64(bz) + bb.minZ,
						maxX: float64(bx) + bb.maxX,
						maxY: float64(by) + bb.maxY,
						maxZ: float64(bz) + bb.maxZ,
					})
				}
			}
		}
	}
	return boxes
}

func blockCollisionRelativeAABBs(blockID, blockMeta int) []axisAlignedBB {
	switch blockID {
	case 0:
		return nil
	case 8, 9, 10, 11:
		return nil
	case 6, 31, 32, 37, 38, 39, 40, 59, 83, 106:
		return nil
	case 66:
		// Translation reference:
		// - net.minecraft.src.BlockRailBase#getCollisionBoundingBoxFromPool(...)
		return nil
	case 70, 72:
		// Translation reference:
		// - net.minecraft.src.BlockBasePressurePlate#getCollisionBoundingBoxFromPool(...)
		return nil
	case 44, 126:
		// Translation reference:
		// - net.minecraft.src.BlockHalfSlab#setBlockBoundsBasedOnState(...)
		if blockMeta&8 != 0 {
			return []axisAlignedBB{{
				minX: 0.0, minY: 0.5, minZ: 0.0,
				maxX: 1.0, maxY: 1.0, maxZ: 1.0,
			}}
		}
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.5, maxZ: 1.0,
		}}
	case 60:
		// Translation reference:
		// - net.minecraft.src.BlockFarmland#getCollisionBoundingBoxFromPool(...)
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 1.0, maxZ: 1.0,
		}}
	case 78:
		// Translation reference:
		// - net.minecraft.src.BlockSnow#getCollisionBoundingBoxFromPool(...)
		// collision height uses metadata*0.125, unlike render/selection bounds.
		height := float64(blockMeta&7) * 0.125
		if height <= 0 {
			return nil
		}
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: height, maxZ: 1.0,
		}}
	case 81:
		// Translation reference:
		// - net.minecraft.src.BlockCactus#getCollisionBoundingBoxFromPool(...)
		return []axisAlignedBB{{
			minX: 0.0625, minY: 0.0, minZ: 0.0625,
			maxX: 0.9375, maxY: 0.9375, maxZ: 0.9375,
		}}
	case 88:
		// Translation reference:
		// - net.minecraft.src.BlockSoulSand#getCollisionBoundingBoxFromPool(...)
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.875, maxZ: 1.0,
		}}
	case 111:
		// Translation reference:
		// - net.minecraft.src.BlockLilyPad#getCollisionBoundingBoxFromPool(...)
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.015625, maxZ: 1.0,
		}}
	}

	if block.BlocksMovement(blockID) || isNormalCubeRenderBlock(blockID) || blockID == 20 {
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 1.0, maxZ: 1.0,
		}}
	}
	return nil
}

func blockRenderRelativeAABB(blockID, blockMeta int) axisAlignedBB {
	// Translation references:
	// - net.minecraft.src.BlockHalfSlab#setBlockBoundsBasedOnState(...)
	// - net.minecraft.src.BlockFarmland constructor bounds
	// - net.minecraft.src.BlockSnow#setBlockBoundsForSnowDepth(...)
	// - net.minecraft.src.BlockCactus#setBlockBounds(...)
	// - net.minecraft.src.BlockSoulSand constructor bounds
	switch blockID {
	case 44, 126:
		if blockMeta&8 != 0 {
			return axisAlignedBB{
				minX: 0.0, minY: 0.5, minZ: 0.0,
				maxX: 1.0, maxY: 1.0, maxZ: 1.0,
			}
		}
		return axisAlignedBB{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.5, maxZ: 1.0,
		}
	case 60:
		return axisAlignedBB{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.9375, maxZ: 1.0,
		}
	case 78:
		depth := blockMeta & 7
		height := float64(2*(1+depth)) / 16.0
		return axisAlignedBB{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: height, maxZ: 1.0,
		}
	case 81:
		return axisAlignedBB{
			minX: 0.0625, minY: 0.0, minZ: 0.0625,
			maxX: 0.9375, maxY: 1.0, maxZ: 0.9375,
		}
	case 88:
		return axisAlignedBB{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.875, maxZ: 1.0,
		}
	default:
		return axisAlignedBB{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 1.0, maxZ: 1.0,
		}
	}
}

func faceFullyOccludedByNeighbor(face int, current, neighbor axisAlignedBB) bool {
	const eps = 1.0e-6
	shifted := neighbor
	switch face {
	case faceDown:
		shifted = shifted.offset(0, -1, 0)
		return shifted.minY <= current.minY+eps &&
			shifted.maxY >= current.minY-eps &&
			shifted.minX <= current.minX+eps &&
			shifted.maxX >= current.maxX-eps &&
			shifted.minZ <= current.minZ+eps &&
			shifted.maxZ >= current.maxZ-eps
	case faceUp:
		shifted = shifted.offset(0, 1, 0)
		return shifted.minY <= current.maxY+eps &&
			shifted.maxY >= current.maxY-eps &&
			shifted.minX <= current.minX+eps &&
			shifted.maxX >= current.maxX-eps &&
			shifted.minZ <= current.minZ+eps &&
			shifted.maxZ >= current.maxZ-eps
	case faceNorth:
		shifted = shifted.offset(0, 0, -1)
		return shifted.minZ <= current.minZ+eps &&
			shifted.maxZ >= current.minZ-eps &&
			shifted.minX <= current.minX+eps &&
			shifted.maxX >= current.maxX-eps &&
			shifted.minY <= current.minY+eps &&
			shifted.maxY >= current.maxY-eps
	case faceSouth:
		shifted = shifted.offset(0, 0, 1)
		return shifted.minZ <= current.maxZ+eps &&
			shifted.maxZ >= current.maxZ-eps &&
			shifted.minX <= current.minX+eps &&
			shifted.maxX >= current.maxX-eps &&
			shifted.minY <= current.minY+eps &&
			shifted.maxY >= current.maxY-eps
	case faceWest:
		shifted = shifted.offset(-1, 0, 0)
		return shifted.minX <= current.minX+eps &&
			shifted.maxX >= current.minX-eps &&
			shifted.minZ <= current.minZ+eps &&
			shifted.maxZ >= current.maxZ-eps &&
			shifted.minY <= current.minY+eps &&
			shifted.maxY >= current.maxY-eps
	case faceEast:
		shifted = shifted.offset(1, 0, 0)
		return shifted.minX <= current.maxX+eps &&
			shifted.maxX >= current.maxX-eps &&
			shifted.minZ <= current.minZ+eps &&
			shifted.maxZ >= current.maxZ-eps &&
			shifted.minY <= current.minY+eps &&
			shifted.maxY >= current.maxY-eps
	default:
		return false
	}
}

func (a *App) pickBlockTarget(snap netclient.StateSnapshot, reach float64) blockTarget {
	if a.session == nil {
		return blockTarget{}
	}

	dirX, dirY, dirZ := lookDirectionFromYawPitch(a.yaw, a.pitch)

	originX := snap.PlayerX
	originY := snap.PlayerY + playerEyeHeight
	originZ := snap.PlayerZ

	visited := make(map[[3]int]struct{}, int(reach/raycastStep)+2)
	best := blockTarget{Dist: reach + 1.0}

	for dist := 0.0; dist <= reach+raycastStep; dist += raycastStep {
		worldX := originX + dirX*dist
		worldY := originY + dirY*dist
		worldZ := originZ + dirZ*dist

		blockX := int(math.Floor(worldX))
		blockY := int(math.Floor(worldY))
		blockZ := int(math.Floor(worldZ))
		posKey := [3]int{blockX, blockY, blockZ}
		if _, seen := visited[posKey]; seen {
			continue
		}
		visited[posKey] = struct{}{}

		id, meta, ok := a.session.BlockAt(blockX, blockY, blockZ)
		if !ok || id == 0 {
			continue
		}

		boxes := a.blockSelectionAABBs(blockX, blockY, blockZ, id, meta)
		for _, bb := range boxes {
			hitDist, hit := rayIntersectAABB(
				originX, originY, originZ,
				dirX, dirY, dirZ,
				bb.minX, bb.minY, bb.minZ,
				bb.maxX, bb.maxY, bb.maxZ,
				reach,
			)
			if !hit {
				continue
			}
			if best.Hit && hitDist >= best.Dist {
				continue
			}
			best = blockTarget{
				X:    blockX,
				Y:    blockY,
				Z:    blockZ,
				Face: blockFaceFromRayHit(originX, originY, originZ, dirX, dirY, dirZ, hitDist, bb),
				Hit:  true,
				Dist: hitDist,
				MinX: bb.minX,
				MinY: bb.minY,
				MinZ: bb.minZ,
				MaxX: bb.maxX,
				MaxY: bb.maxY,
				MaxZ: bb.maxZ,
			}
		}
	}

	if best.Hit {
		return best
	}

	return blockTarget{}
}

func (a *App) blockTargetDistance(snap netclient.StateSnapshot, target blockTarget) float64 {
	if !target.Hit {
		return math.Inf(1)
	}
	if target.Dist > 0 {
		return target.Dist
	}

	ox := snap.PlayerX
	oy := snap.PlayerY + playerEyeHeight
	oz := snap.PlayerZ
	cx := float64(target.X) + 0.5
	cy := float64(target.Y) + 0.5
	cz := float64(target.Z) + 0.5
	dx := cx - ox
	dy := cy - oy
	dz := cz - oz
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (a *App) blockSelectionAABBs(blockX, blockY, blockZ, blockID, blockMeta int) []axisAlignedBB {
	if isLiquidRenderBlock(blockID) {
		// Translation reference:
		// - net.minecraft.src.BlockFluid#getCollisionBoundingBoxFromPool => null
		// - net.minecraft.src.BlockFluid#canCollideCheck(meta, hitIfLiquid)
		// In normal gameplay ray picks should pass through liquid unless explicit liquid check mode.
		return nil
	}

	if blockID == 106 && blockMeta == 0 {
		// Match vine render fallback for legacy chunks where metadata is absent.
		blockMeta = a.inferSingleVineAttachmentMeta(blockX, blockY, blockZ)
	}

	relative := blockSelectionRelativeAABBs(blockID, blockMeta)
	if len(relative) == 0 {
		return nil
	}

	ox := float64(blockX)
	oy := float64(blockY)
	oz := float64(blockZ)
	out := make([]axisAlignedBB, 0, len(relative))
	for _, bb := range relative {
		out = append(out, axisAlignedBB{
			minX: ox + bb.minX,
			minY: oy + bb.minY,
			minZ: oz + bb.minZ,
			maxX: ox + bb.maxX,
			maxY: oy + bb.maxY,
			maxZ: oz + bb.maxZ,
		})
	}
	return out
}

func blockSelectionRelativeAABBs(blockID, blockMeta int) []axisAlignedBB {
	switch blockID {
	case 8, 9, 10, 11:
		return nil
	case 44, 126:
		// Translation reference:
		// - net.minecraft.src.BlockHalfSlab#setBlockBoundsBasedOnState(...)
		if blockMeta&8 != 0 {
			return []axisAlignedBB{{
				minX: 0.0, minY: 0.5, minZ: 0.0,
				maxX: 1.0, maxY: 1.0, maxZ: 1.0,
			}}
		}
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.5, maxZ: 1.0,
		}}
	case 60:
		// Translation reference:
		// - net.minecraft.src.BlockFarmland constructor bounds.
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.9375, maxZ: 1.0,
		}}
	case 66:
		// Translation reference:
		// - net.minecraft.src.BlockRailBase#setBlockBoundsBasedOnState(...)
		railMeta := blockMeta & 7
		railHeight := 0.125
		if railMeta >= 2 && railMeta <= 5 {
			railHeight = 0.625
		}
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: railHeight, maxZ: 1.0,
		}}
	case 70, 72:
		// Translation reference:
		// - net.minecraft.src.BlockBasePressurePlate#setBlockBoundsBasedOnState(...)
		height := 0.0625
		if blockMeta > 0 {
			height = 0.03125
		}
		return []axisAlignedBB{{
			minX: 0.0625, minY: 0.0, minZ: 0.0625,
			maxX: 0.9375, maxY: height, maxZ: 0.9375,
		}}
	case 78:
		// Translation reference:
		// - net.minecraft.src.BlockSnow#setBlockBoundsForSnowDepth(...)
		depth := blockMeta & 7
		height := float64(2*(1+depth)) / 16.0
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: height, maxZ: 1.0,
		}}
	case 81:
		// Translation reference:
		// - net.minecraft.src.BlockCactus#getSelectedBoundingBoxFromPool(...)
		return []axisAlignedBB{{
			minX: 0.0625, minY: 0.0, minZ: 0.0625,
			maxX: 0.9375, maxY: 1.0, maxZ: 0.9375,
		}}
	case 6, 31, 32:
		// Translation reference:
		// - net.minecraft.src.BlockSapling
		// - net.minecraft.src.BlockTallGrass
		// - net.minecraft.src.BlockDeadBush
		return []axisAlignedBB{{
			minX: 0.1, minY: 0.0, minZ: 0.1,
			maxX: 0.9, maxY: 0.8, maxZ: 0.9,
		}}
	case 37, 38, 39, 40:
		// Translation reference:
		// - net.minecraft.src.BlockFlower constructor bounds.
		return []axisAlignedBB{{
			minX: 0.3, minY: 0.0, minZ: 0.3,
			maxX: 0.7, maxY: 0.6, maxZ: 0.7,
		}}
	case 59:
		// Translation reference:
		// - net.minecraft.src.BlockCrops#setBlockBoundsBasedOnState(...)
		age := blockMeta & 7
		height := float64(age*2+2) / 16.0
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: height, maxZ: 1.0,
		}}
	case 83:
		// Translation reference:
		// - net.minecraft.src.BlockReed constructor bounds.
		return []axisAlignedBB{{
			minX: 0.125, minY: 0.0, minZ: 0.125,
			maxX: 0.875, maxY: 1.0, maxZ: 0.875,
		}}
	case 106:
		return []axisAlignedBB{vineSelectionRelativeAABB(blockMeta)}
	case 111:
		// Translation reference:
		// - net.minecraft.src.BlockLilyPad constructor bounds.
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 0.015625, maxZ: 1.0,
		}}
	default:
		return []axisAlignedBB{{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 1.0, maxZ: 1.0,
		}}
	}
}

func vineSelectionRelativeAABB(meta int) axisAlignedBB {
	// Translation reference:
	// - net.minecraft.src.BlockVine#setBlockBoundsBasedOnState(...)
	minX := 1.0
	minY := 1.0
	minZ := 1.0
	maxX := 0.0
	maxY := 0.0
	maxZ := 0.0
	attached := meta > 0

	if meta&2 != 0 {
		if maxX < 0.0625 {
			maxX = 0.0625
		}
		minX = 0.0
		minY = 0.0
		maxY = 1.0
		minZ = 0.0
		maxZ = 1.0
		attached = true
	}
	if meta&8 != 0 {
		if minX > 0.9375 {
			minX = 0.9375
		}
		maxX = 1.0
		minY = 0.0
		maxY = 1.0
		minZ = 0.0
		maxZ = 1.0
		attached = true
	}
	if meta&4 != 0 {
		if maxZ < 0.0625 {
			maxZ = 0.0625
		}
		minZ = 0.0
		minX = 0.0
		maxX = 1.0
		minY = 0.0
		maxY = 1.0
		attached = true
	}
	if meta&1 != 0 {
		if minZ > 0.9375 {
			minZ = 0.9375
		}
		maxZ = 1.0
		minX = 0.0
		maxX = 1.0
		minY = 0.0
		maxY = 1.0
		attached = true
	}
	if !attached {
		minY = math.Min(minY, 0.9375)
		maxY = 1.0
		minX = 0.0
		maxX = 1.0
		minZ = 0.0
		maxZ = 1.0
	}
	if minX >= maxX || minY >= maxY || minZ >= maxZ {
		return axisAlignedBB{
			minX: 0.0, minY: 0.0, minZ: 0.0,
			maxX: 1.0, maxY: 1.0, maxZ: 1.0,
		}
	}
	return axisAlignedBB{
		minX: minX, minY: minY, minZ: minZ,
		maxX: maxX, maxY: maxY, maxZ: maxZ,
	}
}

func blockFaceFromRayHit(ox, oy, oz, dx, dy, dz, dist float64, bb axisAlignedBB) int32 {
	hx := ox + dx*dist
	hy := oy + dy*dist
	hz := oz + dz*dist

	bestDelta := math.Abs(hx - bb.minX)
	bestFace := int32(faceWest)
	candidates := []struct {
		delta float64
		face  int32
	}{
		{delta: math.Abs(hx - bb.maxX), face: faceEast},
		{delta: math.Abs(hy - bb.minY), face: faceDown},
		{delta: math.Abs(hy - bb.maxY), face: faceUp},
		{delta: math.Abs(hz - bb.minZ), face: faceNorth},
		{delta: math.Abs(hz - bb.maxZ), face: faceSouth},
	}
	for _, c := range candidates {
		if c.delta < bestDelta {
			bestDelta = c.delta
			bestFace = c.face
		}
	}

	const eps = 1.0e-4
	if bestDelta <= eps {
		return bestFace
	}

	// Fallback for precision-edge hits.
	absX := math.Abs(dx)
	absY := math.Abs(dy)
	absZ := math.Abs(dz)
	if absX >= absY && absX >= absZ {
		if dx > 0 {
			return faceWest
		}
		return faceEast
	}
	if absY >= absX && absY >= absZ {
		if dy > 0 {
			return faceDown
		}
		return faceUp
	}
	if dz > 0 {
		return faceNorth
	}
	return faceSouth
}

func (a *App) pickEntityTarget(snap netclient.StateSnapshot, reach float64) entityTarget {
	if a.session == nil {
		return entityTarget{}
	}
	entities := a.session.EntitiesSnapshot()
	if len(entities) == 0 {
		return entityTarget{}
	}

	dirX, dirY, dirZ := lookDirectionFromYawPitch(a.yaw, a.pitch)
	originX := snap.PlayerX
	originY := snap.PlayerY + playerEyeHeight
	originZ := snap.PlayerZ

	best := entityTarget{Dist: reach + 1.0}
	for _, ent := range entities {
		width, height := entityCollisionSize(ent)
		half := width * 0.5
		minX := ent.X - half
		minY := ent.Y
		minZ := ent.Z - half
		maxX := ent.X + half
		maxY := ent.Y + height
		maxZ := ent.Z + half

		// Translation reference:
		// - net.minecraft.src.EntityRenderer#getMouseOver() expands hit boxes by collision border size.
		const border = 0.1
		minX -= border
		minY -= border
		minZ -= border
		maxX += border
		maxY += border
		maxZ += border

		dist, hit := rayIntersectAABB(originX, originY, originZ, dirX, dirY, dirZ, minX, minY, minZ, maxX, maxY, maxZ, reach)
		if !hit {
			continue
		}
		if !best.Hit || dist < best.Dist {
			best.Hit = true
			best.Dist = dist
			best.EntityID = ent.EntityID
		}
	}
	return best
}

func entityCollisionSize(ent netclient.EntitySnapshot) (width float64, height float64) {
	switch ent.Type {
	case 0: // player
		return 0.6, 1.8
	case 50, 51, 54, 57, 66, 120: // creeper/skeleton/zombie/pigzombie/witch/villager
		return 0.6, 1.8
	case 52: // spider
		return 1.4, 0.9
	case 55: // slime
		size := float64(ent.SlimeSize)
		if size < 1.0 {
			size = 1.0
		}
		side := 0.6 * size
		return side, side
	case 58: // enderman
		return 0.6, 2.9
	case 65: // bat
		return 0.5, 0.9
	case 90: // pig
		return 0.9, 0.9
	case 91, 92: // sheep/cow
		return 0.9, 1.3
	case 93: // chicken
		return 0.3, 0.7
	case 94: // squid
		return 0.95, 0.95
	case 100: // horse
		return 1.4, 1.6
	default:
		return 0.6, 1.8
	}
}

func rayIntersectAABB(
	ox, oy, oz float64,
	dx, dy, dz float64,
	minX, minY, minZ float64,
	maxX, maxY, maxZ float64,
	maxDist float64,
) (float64, bool) {
	tMin := 0.0
	tMax := maxDist

	axis := func(origin, dir, minB, maxB float64) bool {
		if math.Abs(dir) < 1.0e-9 {
			return origin >= minB && origin <= maxB
		}
		inv := 1.0 / dir
		t1 := (minB - origin) * inv
		t2 := (maxB - origin) * inv
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tMin {
			tMin = t1
		}
		if t2 < tMax {
			tMax = t2
		}
		return tMax >= tMin
	}

	if !axis(ox, dx, minX, maxX) {
		return 0, false
	}
	if !axis(oy, dy, minY, maxY) {
		return 0, false
	}
	if !axis(oz, dz, minZ, maxZ) {
		return 0, false
	}
	if tMax < 0 {
		return 0, false
	}
	if tMin < 0 {
		tMin = tMax
	}
	if tMin < 0 || tMin > maxDist {
		return 0, false
	}
	return tMin, true
}

func (a *App) drainSessionEvents() {
	if a.chatClosed || a.eventsCh == nil {
		return
	}
	for {
		select {
		case ev, ok := <-a.eventsCh:
			if !ok {
				a.chatClosed = true
				return
			}
			switch ev.Type {
			case netclient.EventChat:
				a.addChatLine(ev.Message, 0xFFFFFF)
			case netclient.EventSystem:
				a.addChatLine("[system] "+ev.Message, 0xA0A0A0)
			case netclient.EventSound:
				if ev.SoundName != "" {
					vol := float64(ev.SoundVolume)
					if vol <= 0 {
						vol = 1.0
					}
					vol *= clampFloat64(a.soundVolume, 0.0, 1.0)
					pitch := float64(ev.SoundPitch)
					if pitch <= 0 {
						pitch = 1.0
					}
					if vol > 0 {
						audio.PlaySoundKey(ev.SoundName, vol, pitch)
					}
				}
			case netclient.EventKick:
				a.addChatLine("[kick] "+ev.Message, 0xFF5555)
			case netclient.EventDisconnect:
				a.addChatLine("[disconnect] "+ev.Message, 0xFF5555)
			default:
				a.addChatLine(ev.Message, 0xFFFFFF)
			}
		default:
			return
		}
	}
}

func (a *App) addChatLine(msg string, color int) {
	if msg == "" {
		return
	}
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	a.chatLines = append(a.chatLines, chatLine{
		Message: msg,
		AddedAt: time.Now(),
		Color:   color,
	})
	if len(a.chatLines) > 200 {
		a.chatLines = a.chatLines[len(a.chatLines)-200:]
	}
}

// Packet15 direction mapping from clicked-face side:
// 0=bottom, 1=top, 2=north(-z), 3=south(+z), 4=west(-x), 5=east(+x)
func blockFaceFromStep(prevX, prevY, prevZ, hitX, hitY, hitZ int) int32 {
	switch {
	case hitX > prevX:
		return 4
	case hitX < prevX:
		return 5
	case hitY > prevY:
		return 0
	case hitY < prevY:
		return 1
	case hitZ > prevZ:
		return 2
	case hitZ < prevZ:
		return 3
	default:
		return 1
	}
}

func begin2D(width, height int) {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, float64(width), float64(height), 0, -1, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
}

func drawSolidRect(x1, y1, x2, y2 int, color int) {
	a := float32((color>>24)&0xFF) / 255.0
	r := float32((color>>16)&0xFF) / 255.0
	g := float32((color>>8)&0xFF) / 255.0
	b := float32(color&0xFF) / 255.0

	gl.Disable(gl.TEXTURE_2D)
	gl.Color4f(r, g, b, a)
	gl.Begin(gl.QUADS)
	gl.Vertex2f(float32(x1), float32(y1))
	gl.Vertex2f(float32(x2), float32(y1))
	gl.Vertex2f(float32(x2), float32(y2))
	gl.Vertex2f(float32(x1), float32(y2))
	gl.End()
	gl.Enable(gl.TEXTURE_2D)
	gl.Color4f(1, 1, 1, 1)
}

func drawGradientRect(x1, y1, x2, y2 int, topColor, bottomColor int) {
	topA := float32((topColor>>24)&0xFF) / 255.0
	topR := float32((topColor>>16)&0xFF) / 255.0
	topG := float32((topColor>>8)&0xFF) / 255.0
	topB := float32(topColor&0xFF) / 255.0

	botA := float32((bottomColor>>24)&0xFF) / 255.0
	botR := float32((bottomColor>>16)&0xFF) / 255.0
	botG := float32((bottomColor>>8)&0xFF) / 255.0
	botB := float32(bottomColor&0xFF) / 255.0

	gl.Disable(gl.TEXTURE_2D)
	gl.ShadeModel(gl.SMOOTH)
	gl.Begin(gl.QUADS)
	gl.Color4f(topR, topG, topB, topA)
	gl.Vertex2f(float32(x1), float32(y1))
	gl.Vertex2f(float32(x2), float32(y1))
	gl.Color4f(botR, botG, botB, botA)
	gl.Vertex2f(float32(x2), float32(y2))
	gl.Vertex2f(float32(x1), float32(y2))
	gl.End()
	gl.ShadeModel(gl.FLAT)
	gl.Enable(gl.TEXTURE_2D)
	gl.Color4f(1, 1, 1, 1)
}

func facingFromYaw(yaw float64) string {
	v := int(math.Floor(yaw*4.0/360.0 + 0.5))
	v &= 3
	switch v {
	case 0:
		return "South (+Z)"
	case 1:
		return "West (-X)"
	case 2:
		return "North (-Z)"
	default:
		return "East (+X)"
	}
}

func drawCube(x, y, z, r, g, b float32) {
	drawCubeFaces(x, y, z, r, g, b, visibleFaces{
		Down:  true,
		Up:    true,
		North: true,
		South: true,
		West:  true,
		East:  true,
	})
}

func drawCubeFaces(x, y, z, r, g, b float32, faces visibleFaces) {
	x2 := x + 1
	y2 := y + 1
	z2 := z + 1

	bright := func(v float32) float32 {
		return float32(math.Min(float64(v+0.18), 1.0))
	}
	dim := func(v float32) float32 {
		return float32(math.Max(float64(v-0.18), 0.0))
	}

	gl.Begin(gl.QUADS)

	if faces.Up {
		gl.Color3f(bright(r), bright(g), bright(b))
		gl.Vertex3f(x, y2, z)
		gl.Vertex3f(x, y2, z2)
		gl.Vertex3f(x2, y2, z2)
		gl.Vertex3f(x2, y2, z)
	}

	if faces.Down {
		gl.Color3f(dim(r), dim(g), dim(b))
		gl.Vertex3f(x, y, z)
		gl.Vertex3f(x2, y, z)
		gl.Vertex3f(x2, y, z2)
		gl.Vertex3f(x, y, z2)
	}

	if faces.North {
		gl.Color3f(r, g, b)
		gl.Vertex3f(x2, y, z)
		gl.Vertex3f(x, y, z)
		gl.Vertex3f(x, y2, z)
		gl.Vertex3f(x2, y2, z)
	}

	if faces.South {
		gl.Color3f(r, g, b)
		gl.Vertex3f(x, y, z2)
		gl.Vertex3f(x2, y, z2)
		gl.Vertex3f(x2, y2, z2)
		gl.Vertex3f(x, y2, z2)
	}

	if faces.West {
		gl.Color3f(dim(r), dim(g), dim(b))
		gl.Vertex3f(x, y, z)
		gl.Vertex3f(x, y, z2)
		gl.Vertex3f(x, y2, z2)
		gl.Vertex3f(x, y2, z)
	}

	if faces.East {
		gl.Color3f(r, g, b)
		gl.Vertex3f(x2, y, z2)
		gl.Vertex3f(x2, y, z)
		gl.Vertex3f(x2, y2, z)
		gl.Vertex3f(x2, y2, z2)
	}

	gl.End()
}

func drawBlockOutline(x, y, z float32) {
	drawBlockOutlineAABB(x, y, z, x+1, y+1, z+1)
}

func drawBlockOutlineAABB(minX, minY, minZ, maxX, maxY, maxZ float32) {
	eps := float32(0.002)
	x1 := minX - eps
	y1 := minY - eps
	z1 := minZ - eps
	x2 := maxX + eps
	y2 := maxY + eps
	z2 := maxZ + eps

	gl.Disable(gl.TEXTURE_2D)
	gl.LineWidth(2)
	gl.Color3f(0.02, 0.02, 0.02)
	gl.Begin(gl.LINES)
	gl.Vertex3f(x1, y1, z1)
	gl.Vertex3f(x2, y1, z1)
	gl.Vertex3f(x2, y1, z1)
	gl.Vertex3f(x2, y1, z2)
	gl.Vertex3f(x2, y1, z2)
	gl.Vertex3f(x1, y1, z2)
	gl.Vertex3f(x1, y1, z2)
	gl.Vertex3f(x1, y1, z1)

	gl.Vertex3f(x1, y2, z1)
	gl.Vertex3f(x2, y2, z1)
	gl.Vertex3f(x2, y2, z1)
	gl.Vertex3f(x2, y2, z2)
	gl.Vertex3f(x2, y2, z2)
	gl.Vertex3f(x1, y2, z2)
	gl.Vertex3f(x1, y2, z2)
	gl.Vertex3f(x1, y2, z1)

	gl.Vertex3f(x1, y1, z1)
	gl.Vertex3f(x1, y2, z1)
	gl.Vertex3f(x2, y1, z1)
	gl.Vertex3f(x2, y2, z1)
	gl.Vertex3f(x2, y1, z2)
	gl.Vertex3f(x2, y2, z2)
	gl.Vertex3f(x1, y1, z2)
	gl.Vertex3f(x1, y2, z2)
	gl.End()
	gl.Enable(gl.TEXTURE_2D)
}

func colorForBlock(id int) (float32, float32, float32) {
	switch id {
	case 2:
		return 0.38, 0.72, 0.22
	case 3:
		return 0.49, 0.36, 0.23
	case 12:
		return 0.84, 0.78, 0.47
	case 8, 9:
		return 0.20, 0.35, 0.90
	case 10, 11:
		return 0.90, 0.36, 0.12
	case 17:
		return 0.55, 0.43, 0.28
	case 18:
		return 0.22, 0.60, 0.20
	case 20:
		return 0.70, 0.90, 0.95
	default:
		return 0.62, 0.62, 0.62
	}
}

func setPerspective(fovY, aspect, near, far float64) {
	top := near * math.Tan(fovY*math.Pi/360.0)
	bottom := -top
	right := top * aspect
	left := -right
	gl.Frustum(left, right, bottom, top, near, far)
}

// Translation reference:
// - net.minecraft.client.Minecraft.getLimitFramerate()
// - net.minecraft.client.Minecraft.runGameLoop() -> Display.sync(...)
// - net.minecraft.client.renderer.EntityRenderer.performanceToFps(int)
func (a *App) currentFrameCap() int {
	// In vanilla, GuiMainMenu forces the "Power saver" framerate mode.
	if a.mainMenu {
		return performanceModeToFPS(2)
	}
	return performanceModeToFPS(a.limitFramerateMode)
}

func (a *App) syncFrameRate(frameStart time.Time) {
	capFPS := a.currentFrameCap()
	if capFPS <= 0 {
		return
	}
	frameDur := time.Second / time.Duration(capFPS)
	target := frameStart.Add(frameDur)
	for {
		remain := time.Until(target)
		if remain <= 0 {
			return
		}
		// Similar to LWJGL Display.sync(): sleep most of the wait budget,
		// then yield in the final small slice for steadier pacing.
		if remain > 2*time.Millisecond {
			time.Sleep(remain - time.Millisecond)
			continue
		}
		runtime.Gosched()
	}
}

func performanceModeToFPS(mode int) int {
	// Translation reference:
	// - net.minecraft.client.renderer.EntityRenderer.performanceToFps(int)
	switch mode {
	case 0:
		return 200 // Max FPS
	case 2:
		return 35 // Power saver
	default:
		return 120 // Balanced (default in 1.6.4 options)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func discoverOptionsPath(assetsRoot string) string {
	assetsRoot = filepath.Clean(assetsRoot)
	if strings.EqualFold(filepath.Base(assetsRoot), "minecraft") {
		assetsDir := filepath.Dir(assetsRoot)
		if strings.EqualFold(filepath.Base(assetsDir), "assets") {
			return filepath.Join(filepath.Dir(assetsDir), "options.txt")
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "options.txt")
	}
	return "options.txt"
}

// Translation reference:
// - net.minecraft.src.GameSettings.loadOptions()
func (a *App) loadOptionsFile() {
	if a.optionsPath == "" {
		return
	}
	if len(a.keyBindings) == 0 {
		a.initDefaultKeyBindings()
	}
	f, err := os.Open(a.optionsPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Printf("gui options warning: load %s failed: %v\n", a.optionsPath, err)
		}
		return
	}
	defer f.Close()

	if a.optionsKV == nil {
		a.optionsKV = make(map[string]string)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		a.optionsKV[key] = value
		switch key {
		case "music":
			if v, parseErr := strconv.ParseFloat(value, 64); parseErr == nil {
				a.musicVolume = clampFloat64(v, 0.0, 1.0)
			}
		case "sound":
			if v, parseErr := strconv.ParseFloat(value, 64); parseErr == nil {
				a.soundVolume = clampFloat64(v, 0.0, 1.0)
			}
		case "fov":
			if v, parseErr := strconv.ParseFloat(value, 64); parseErr == nil {
				a.fovSetting = clampFloat64(v, 0.0, 1.0)
			}
		case "invertYMouse":
			if v, parseErr := strconv.ParseBool(value); parseErr == nil {
				a.invertMouse = v
			}
		case "viewDistance":
			if v, parseErr := strconv.Atoi(value); parseErr == nil {
				a.renderDistance = normalizeRenderDistanceMode(v)
			}
		case "guiScale":
			if v, parseErr := strconv.Atoi(value); parseErr == nil {
				a.guiScaleMode = clampInt(v, 0, len(guiScaleModeNames)-1)
			}
		case "bobView":
			if v, parseErr := strconv.ParseBool(value); parseErr == nil {
				a.viewBobbing = v
			}
		case "mouseSensitivity":
			if v, parseErr := strconv.ParseFloat(value, 64); parseErr == nil {
				a.mouseSens = clampFloat64(v, 0.0, 1.0)
			}
		case "fpsLimit":
			if v, parseErr := strconv.Atoi(value); parseErr == nil {
				a.limitFramerateMode = clampInt(v, 0, 2)
			}
		case "difficulty":
			if v, parseErr := strconv.Atoi(value); parseErr == nil {
				a.optionDifficulty = v & 3
			}
		case "fancyGraphics":
			if v, parseErr := strconv.ParseBool(value); parseErr == nil {
				a.fancyGraphics = v
			}
		case "clouds":
			if v, parseErr := strconv.ParseBool(value); parseErr == nil {
				a.cloudsEnabled = v
			}
		}
		if strings.HasPrefix(key, "key_") {
			desc := strings.TrimPrefix(key, "key_")
			if idx := a.keyBindingIndexByDescription(desc); idx >= 0 {
				if v, parseErr := strconv.Atoi(value); parseErr == nil {
					a.keyBindings[idx].Code = v
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("gui options warning: read %s failed: %v\n", a.optionsPath, err)
	}
}

// Translation reference:
// - net.minecraft.src.GameSettings.saveOptions()
func (a *App) saveOptionsFile() {
	if a.optionsPath == "" {
		return
	}
	if len(a.keyBindings) == 0 {
		a.initDefaultKeyBindings()
	}
	if a.optionsKV == nil {
		a.optionsKV = make(map[string]string)
	}
	a.optionsKV["music"] = strconv.FormatFloat(clampFloat64(a.musicVolume, 0.0, 1.0), 'f', 6, 64)
	a.optionsKV["sound"] = strconv.FormatFloat(clampFloat64(a.soundVolume, 0.0, 1.0), 'f', 6, 64)
	a.optionsKV["fov"] = strconv.FormatFloat(clampFloat64(a.fovSetting, 0.0, 1.0), 'f', 6, 64)
	a.optionsKV["invertYMouse"] = strconv.FormatBool(a.invertMouse)
	a.optionsKV["viewDistance"] = strconv.Itoa(normalizeRenderDistanceMode(a.renderDistance))
	a.optionsKV["guiScale"] = strconv.Itoa(clampInt(a.guiScaleMode, 0, len(guiScaleModeNames)-1))
	a.optionsKV["bobView"] = strconv.FormatBool(a.viewBobbing)
	a.optionsKV["mouseSensitivity"] = strconv.FormatFloat(clampFloat64(a.mouseSens, 0.0, 1.0), 'f', 6, 64)
	a.optionsKV["fpsLimit"] = strconv.Itoa(clampInt(a.limitFramerateMode, 0, 2))
	a.optionsKV["difficulty"] = strconv.Itoa(a.optionDifficulty & 3)
	a.optionsKV["fancyGraphics"] = strconv.FormatBool(a.fancyGraphics)
	a.optionsKV["clouds"] = strconv.FormatBool(a.cloudsEnabled)
	for i := range a.keyBindings {
		a.optionsKV["key_"+a.keyBindings[i].Description] = strconv.Itoa(a.keyBindings[i].Code)
	}

	keys := make([]string, 0, len(a.optionsKV))
	for k := range a.optionsKV {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if err := os.MkdirAll(filepath.Dir(a.optionsPath), 0o755); err != nil {
		fmt.Printf("gui options warning: ensure options dir failed: %v\n", err)
		return
	}
	tmpPath := a.optionsPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		fmt.Printf("gui options warning: create temp options failed: %v\n", err)
		return
	}
	writer := bufio.NewWriter(file)
	for _, key := range keys {
		if _, err := fmt.Fprintf(writer, "%s:%s\n", key, a.optionsKV[key]); err != nil {
			_ = file.Close()
			_ = os.Remove(tmpPath)
			fmt.Printf("gui options warning: write options failed: %v\n", err)
			return
		}
	}
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		fmt.Printf("gui options warning: flush options failed: %v\n", err)
		return
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		fmt.Printf("gui options warning: close temp options failed: %v\n", err)
		return
	}
	if err := os.Rename(tmpPath, a.optionsPath); err != nil {
		_ = os.Remove(tmpPath)
		fmt.Printf("gui options warning: replace options failed: %v\n", err)
		return
	}
}

func discoverAssetsRoot() string {
	defaultRoot := filepath.Join("assets", "minecraft")
	candidates := make([]string, 0, 32)

	appendSearchRoots := func(base string) {
		if base == "" {
			return
		}
		cur := base
		for i := 0; i < 8; i++ {
			candidates = append(candidates, filepath.Join(cur, "assets", "minecraft"))
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}

	if wd, err := os.Getwd(); err == nil {
		appendSearchRoots(wd)
	}
	if exePath, err := os.Executable(); err == nil {
		appendSearchRoots(filepath.Dir(exePath))
	}
	candidates = append(candidates, defaultRoot)

	seen := make(map[string]struct{}, len(candidates))
	uniq := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		uniq = append(uniq, c)
	}

	hasMainMenuAssets := func(root string) bool {
		required := []string{
			filepath.Join(root, "textures", "gui", "title", "background", "panorama_0.png"),
			filepath.Join(root, "textures", "gui", "widgets.png"),
			filepath.Join(root, "textures", "font", "ascii.png"),
		}
		for _, p := range required {
			st, err := os.Stat(p)
			if err != nil || st.IsDir() {
				return false
			}
		}
		return true
	}

	for _, root := range uniq {
		if hasMainMenuAssets(root) {
			return root
		}
	}
	for _, root := range uniq {
		if st, err := os.Stat(filepath.Join(root, "textures")); err == nil && st.IsDir() {
			return root
		}
	}
	return defaultRoot
}

func discoverFontCharsPath(assetsRoot string) string {
	defaultPath := filepath.Join("assets", "font.txt")
	candidates := make([]string, 0, 24)

	appendSearch := func(base string) {
		if base == "" {
			return
		}
		cur := base
		for i := 0; i < 8; i++ {
			candidates = append(candidates, filepath.Join(cur, "assets", "font.txt"))
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}
	candidates = append(candidates, defaultPath)
	candidates = append(candidates, filepath.Join(filepath.Dir(assetsRoot), "font.txt"))
	if wd, err := os.Getwd(); err == nil {
		appendSearch(wd)
	}
	if exePath, err := os.Executable(); err == nil {
		appendSearch(filepath.Dir(exePath))
	}

	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return defaultPath
}

func isOpaqueRenderBlock(id int) bool {
	switch id {
	case 8, 9, 10, 11, 18, 20, 31, 32, 37, 38, 39, 40, 44, 50, 51, 59, 60, 63, 64, 65, 66, 68, 69, 70, 71, 72, 75, 76, 77, 78, 79, 81, 83, 85, 88, 90, 106, 111, 126, 127:
		return false
	default:
		return true
	}
}

func isWaterRenderBlock(id int) bool {
	return id == 8 || id == 9
}

func isLavaRenderBlock(id int) bool {
	return id == 10 || id == 11
}

func isLiquidRenderBlock(id int) bool {
	return isWaterRenderBlock(id) || isLavaRenderBlock(id)
}

func isSameLiquidMaterial(a, b int) bool {
	if isWaterRenderBlock(a) {
		return isWaterRenderBlock(b)
	}
	if isLavaRenderBlock(a) {
		return isLavaRenderBlock(b)
	}
	return false
}

func isNormalCubeRenderBlock(id int) bool {
	return isOpaqueRenderBlock(id)
}

func isFlatPlantRenderBlock(id int) bool {
	switch id {
	case 111:
		return true
	default:
		return false
	}
}

func isCrossedPlantRenderBlock(id int) bool {
	switch id {
	case 6, 31, 32, 37, 38, 39, 40, 59, 83:
		return true
	default:
		return false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
