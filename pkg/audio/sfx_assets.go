package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jfreymuth/oggvorbis"
)

type soundLibrary struct {
	mu    sync.Mutex
	rng   *mrand.Rand
	byKey map[string][]string
}

var (
	soundLibMu   sync.RWMutex
	soundLib     *soundLibrary
	soundLibDiag string
	pcmCache     sync.Map // map[string][]byte
)

// Translation source: net.minecraft.src.Block static registrations (MCP 1.6.4)
// setStepSound(sound*Footstep) for block IDs.
var blockStepSoundByID = map[int]string{
	1:   "soundStoneFootstep",
	2:   "soundGrassFootstep",
	3:   "soundGravelFootstep",
	4:   "soundStoneFootstep",
	5:   "soundWoodFootstep",
	6:   "soundGrassFootstep",
	7:   "soundStoneFootstep",
	12:  "soundSandFootstep",
	13:  "soundGravelFootstep",
	14:  "soundStoneFootstep",
	15:  "soundStoneFootstep",
	16:  "soundStoneFootstep",
	17:  "soundWoodFootstep",
	18:  "soundGrassFootstep",
	19:  "soundGrassFootstep",
	20:  "soundGlassFootstep",
	21:  "soundStoneFootstep",
	22:  "soundStoneFootstep",
	23:  "soundStoneFootstep",
	24:  "soundStoneFootstep",
	27:  "soundMetalFootstep",
	28:  "soundMetalFootstep",
	31:  "soundGrassFootstep",
	32:  "soundGrassFootstep",
	35:  "soundClothFootstep",
	37:  "soundGrassFootstep",
	38:  "soundGrassFootstep",
	39:  "soundGrassFootstep",
	40:  "soundGrassFootstep",
	41:  "soundMetalFootstep",
	42:  "soundMetalFootstep",
	43:  "soundStoneFootstep",
	44:  "soundStoneFootstep",
	45:  "soundStoneFootstep",
	46:  "soundGrassFootstep",
	47:  "soundWoodFootstep",
	48:  "soundStoneFootstep",
	49:  "soundStoneFootstep",
	50:  "soundWoodFootstep",
	51:  "soundWoodFootstep",
	52:  "soundMetalFootstep",
	54:  "soundWoodFootstep",
	55:  "soundPowderFootstep",
	56:  "soundStoneFootstep",
	57:  "soundMetalFootstep",
	58:  "soundWoodFootstep",
	60:  "soundGravelFootstep",
	61:  "soundStoneFootstep",
	62:  "soundStoneFootstep",
	63:  "soundWoodFootstep",
	64:  "soundWoodFootstep",
	65:  "soundLadderFootstep",
	66:  "soundMetalFootstep",
	68:  "soundWoodFootstep",
	69:  "soundWoodFootstep",
	70:  "soundStoneFootstep",
	71:  "soundMetalFootstep",
	72:  "soundWoodFootstep",
	73:  "soundStoneFootstep",
	74:  "soundStoneFootstep",
	75:  "soundWoodFootstep",
	76:  "soundWoodFootstep",
	77:  "soundStoneFootstep",
	78:  "soundSnowFootstep",
	79:  "soundGlassFootstep",
	80:  "soundSnowFootstep",
	81:  "soundClothFootstep",
	82:  "soundGravelFootstep",
	83:  "soundGrassFootstep",
	84:  "soundStoneFootstep",
	85:  "soundWoodFootstep",
	86:  "soundWoodFootstep",
	87:  "soundStoneFootstep",
	88:  "soundSandFootstep",
	89:  "soundGlassFootstep",
	90:  "soundGlassFootstep",
	91:  "soundWoodFootstep",
	92:  "soundClothFootstep",
	93:  "soundWoodFootstep",
	94:  "soundWoodFootstep",
	95:  "soundWoodFootstep",
	96:  "soundWoodFootstep",
	98:  "soundStoneFootstep",
	99:  "soundWoodFootstep",
	100: "soundWoodFootstep",
	101: "soundMetalFootstep",
	102: "soundGlassFootstep",
	103: "soundWoodFootstep",
	104: "soundWoodFootstep",
	105: "soundWoodFootstep",
	106: "soundGrassFootstep",
	107: "soundWoodFootstep",
	110: "soundGrassFootstep",
	111: "soundGrassFootstep",
	112: "soundStoneFootstep",
	113: "soundStoneFootstep",
	120: "soundGlassFootstep",
	121: "soundStoneFootstep",
	122: "soundStoneFootstep",
	123: "soundGlassFootstep",
	124: "soundGlassFootstep",
	125: "soundWoodFootstep",
	126: "soundWoodFootstep",
	127: "soundWoodFootstep",
	129: "soundStoneFootstep",
	130: "soundStoneFootstep",
	133: "soundMetalFootstep",
	140: "soundPowderFootstep",
	143: "soundWoodFootstep",
	144: "soundStoneFootstep",
	145: "soundAnvilFootstep",
	146: "soundWoodFootstep",
	147: "soundWoodFootstep",
	148: "soundWoodFootstep",
	149: "soundWoodFootstep",
	150: "soundWoodFootstep",
	151: "soundWoodFootstep",
	152: "soundMetalFootstep",
	153: "soundStoneFootstep",
	154: "soundWoodFootstep",
	155: "soundStoneFootstep",
	157: "soundMetalFootstep",
	158: "soundStoneFootstep",
	159: "soundStoneFootstep",
	170: "soundGrassFootstep",
	171: "soundClothFootstep",
	172: "soundStoneFootstep",
	173: "soundStoneFootstep",
}

func initSoundLibrary(assetsRoot string) {
	lib := buildSoundLibrary(assetsRoot)
	soundLibMu.Lock()
	soundLib = lib
	soundLibMu.Unlock()
}

func soundLibraryDiagnostic() string {
	soundLibMu.RLock()
	defer soundLibMu.RUnlock()
	return soundLibDiag
}

func buildSoundLibrary(assetsRoot string) *soundLibrary {
	roots := discoverSoundRoots(assetsRoot)
	lib := &soundLibrary{
		rng:   mrand.New(mrand.NewSource(time.Now().UnixNano())),
		byKey: make(map[string][]string),
	}

	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".ogg") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			indexSoundFile(lib.byKey, rel, path)
			return nil
		})
	}

	if len(lib.byKey) == 0 {
		soundLibDiag = fmt.Sprintf(
			"audio: no ogg assets found; copy 1.6.4 sound resources into assets (checked %d roots)",
			len(roots),
		)
	} else {
		soundLibDiag = fmt.Sprintf("audio: loaded %d sound keys", len(lib.byKey))
	}
	return lib
}

func discoverSoundRoots(assetsRoot string) []string {
	var out []string
	seen := make(map[string]struct{})
	addIfDir := func(path string) {
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		st, err := os.Stat(path)
		if err != nil || !st.IsDir() {
			return
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	addSoundSubdirs := func(base string) {
		addIfDir(filepath.Join(base, "sound"))
		addIfDir(filepath.Join(base, "records"))
		addIfDir(filepath.Join(base, "music"))
	}

	if assetsRoot != "" {
		addSoundSubdirs(assetsRoot)
		parent := filepath.Dir(assetsRoot)
		addSoundSubdirs(parent)
		addSoundSubdirs(filepath.Join(parent, "resources"))
	}

	if appData := os.Getenv("APPDATA"); appData != "" {
		mc := filepath.Join(appData, ".minecraft")
		addSoundSubdirs(filepath.Join(mc, "resources"))
		addSoundSubdirs(filepath.Join(mc, "assets", "virtual", "legacy"))
	}

	sort.Strings(out)
	return out
}

func indexSoundFile(byKey map[string][]string, relPath, fullPath string) {
	key := normalizeSoundKey(relPath)
	if key == "" {
		return
	}
	var keys []string
	keys = append(keys, key, strings.ReplaceAll(key, "/", "."))
	base := trimVariantSuffix(key)
	keys = append(keys, base, strings.ReplaceAll(base, "/", "."))

	for _, k := range keys {
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" {
			continue
		}
		byKey[k] = appendUnique(byKey[k], fullPath)
	}
}

func normalizeSoundKey(path string) string {
	path = strings.ToLower(filepath.ToSlash(path))
	path = strings.TrimSuffix(path, ".ogg")
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "sound/")
	path = strings.TrimPrefix(path, "sounds/")
	path = strings.TrimPrefix(path, "newsound/")
	path = strings.TrimPrefix(path, "records/")
	path = strings.TrimPrefix(path, "music/")
	path = strings.Trim(path, "/")
	return path
}

func trimVariantSuffix(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return key
	}
	last := parts[len(parts)-1]
	i := len(last)
	for i > 0 {
		c := last[i-1]
		if c < '0' || c > '9' {
			break
		}
		i--
	}
	if i <= 0 {
		return key
	}
	parts[len(parts)-1] = last[:i]
	return strings.Join(parts, "/")
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

func playSoundKeys(keys []string, volume, pitch float64) bool {
	soundLibMu.RLock()
	lib := soundLib
	soundLibMu.RUnlock()
	if lib == nil || len(lib.byKey) == 0 {
		return false
	}

	for _, rawKey := range keys {
		key := normalizeSoundKey(rawKey)
		paths := lib.byKey[key]
		if len(paths) == 0 {
			key = strings.ReplaceAll(key, "/", ".")
			paths = lib.byKey[key]
		}
		if len(paths) == 0 {
			continue
		}

		lib.mu.Lock()
		chosen := paths[lib.rng.Intn(len(paths))]
		lib.mu.Unlock()

		pcm, ok := pcmForOggFile(chosen)
		if !ok {
			continue
		}
		if pitch > 0 && math.Abs(pitch-1.0) > 1.0e-3 {
			pcm = pitchShiftPCM16Stereo(pcm, pitch)
		}
		if playPCMBytes(pcm, volume) {
			return true
		}
	}
	return false
}

func digSoundForBlock(blockID int) ([]string, float64, float64) {
	stepTag, ok := blockStepSoundByID[blockID]
	if !ok {
		// Block constructor default in 1.6.4 is soundPowderFootstep.
		stepTag = "soundPowderFootstep"
	}
	def := stepSoundDef(stepTag)
	if def.breakKey == "" {
		return nil, 0, 0
	}
	// Translation source:
	// - net.minecraft.src.PlayerControllerMP#onPlayerDestroyBlock
	// - net.minecraft.src.RenderGlobal#playAuxSFX case 2001:
	// sndManager.playSound(stepSound.getBreakSound(), ..., (vol+1)/2, pitch*0.8)
	// Current rewrite destroys targeted block on click, so use break-sound formula.
	return []string{def.breakKey}, (def.volume + 1.0) / 2.0, def.pitch * 0.8
}

func placeSoundForBlock(blockID int) ([]string, float64, float64) {
	stepTag, ok := blockStepSoundByID[blockID]
	if !ok {
		stepTag = "soundPowderFootstep"
	}
	def := stepSoundDef(stepTag)
	if len(def.placeKeys) == 0 {
		return nil, 0, 0
	}
	// Translation source:
	// - net.minecraft.src.ItemBlock#onItemUse / ItemReed / ItemSlab / ItemSnow
	// world.playSoundEffect(..., stepSound.getPlaceSound(), (vol+1)/2, pitch*0.8)
	return def.placeKeys, (def.volume + 1.0) / 2.0, def.pitch * 0.8
}

type stepSoundInfo struct {
	volume    float64
	pitch     float64
	breakKey  string
	placeKeys []string
}

func stepSoundDef(stepTag string) stepSoundInfo {
	switch stepTag {
	case "soundGlassFootstep":
		// StepSoundStone: break=random.glass, place=step.stone
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "random.glass",
			placeKeys: []string{"step.stone"},
		}
	case "soundAnvilFootstep":
		// StepSoundAnvil: break=dig.stone, place=random.anvil_land
		return stepSoundInfo{
			volume:    0.3,
			pitch:     1.0,
			breakKey:  "dig.stone",
			placeKeys: []string{"random.anvil_land"},
		}
	case "soundLadderFootstep":
		// StepSoundSand#getBreakSound() -> dig.wood, and place falls back to break.
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.wood",
			placeKeys: []string{"dig.wood"},
		}
	case "soundWoodFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.wood",
			placeKeys: []string{"dig.wood"},
		}
	case "soundGravelFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.gravel",
			placeKeys: []string{"dig.gravel"},
		}
	case "soundGrassFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.grass",
			placeKeys: []string{"dig.grass"},
		}
	case "soundSandFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.sand",
			placeKeys: []string{"dig.sand"},
		}
	case "soundSnowFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.snow",
			placeKeys: []string{"dig.snow"},
		}
	case "soundClothFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.cloth",
			placeKeys: []string{"dig.cloth"},
		}
	case "soundMetalFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.5,
			breakKey:  "dig.stone",
			placeKeys: []string{"dig.stone"},
		}
	case "soundStoneFootstep", "soundPowderFootstep":
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.stone",
			placeKeys: []string{"dig.stone"},
		}
	default:
		return stepSoundInfo{
			volume:    1.0,
			pitch:     1.0,
			breakKey:  "dig.stone",
			placeKeys: []string{"dig.stone"},
		}
	}
}

func pcmForOggFile(path string) ([]byte, bool) {
	if v, ok := pcmCache.Load(path); ok {
		if data, ok2 := v.([]byte); ok2 && len(data) > 0 {
			return data, true
		}
	}
	data, err := decodeOggToPCM(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	pcmCache.Store(path, data)
	return data, true
}

func decodeOggToPCM(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec, err := oggvorbis.NewReader(f)
	if err != nil {
		return nil, err
	}
	srcChannels := dec.Channels()
	srcRate := dec.SampleRate()
	if srcChannels <= 0 || srcRate <= 0 {
		return nil, fmt.Errorf("invalid ogg stream: channels=%d rate=%d", srcChannels, srcRate)
	}

	readBuf := make([]float32, 8192*maxInt(srcChannels, 1))
	samples := make([]float32, 0, 16384)
	for {
		n, rerr := dec.Read(readBuf)
		if n > 0 {
			samples = append(samples, readBuf[:n]...)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return nil, rerr
		}
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("empty ogg stream")
	}

	frames := len(samples) / srcChannels
	if frames <= 0 {
		return nil, fmt.Errorf("invalid decoded frame count")
	}

	targetFrames := frames
	if srcRate != otoSampleRate {
		targetFrames = int(math.Round(float64(frames) * float64(otoSampleRate) / float64(srcRate)))
		if targetFrames < 1 {
			targetFrames = 1
		}
	}

	out := make([]byte, targetFrames*otoChannels*2)
	for i := 0; i < targetFrames; i++ {
		srcFrame := i
		if srcRate != otoSampleRate {
			srcFrame = int(float64(i) * float64(srcRate) / float64(otoSampleRate))
		}
		if srcFrame >= frames {
			srcFrame = frames - 1
		}
		base := srcFrame * srcChannels
		l := clampPCM(samples[base])
		r := l
		if srcChannels > 1 {
			r = clampPCM(samples[base+1])
		}

		off := i * 4
		binary.LittleEndian.PutUint16(out[off:], uint16(int16(l*32767.0)))
		binary.LittleEndian.PutUint16(out[off+2:], uint16(int16(r*32767.0)))
	}
	return out, nil
}

func clampPCM(v float32) float32 {
	if v < -1 {
		return -1
	}
	if v > 1 {
		return 1
	}
	return v
}

func pitchShiftPCM16Stereo(src []byte, pitch float64) []byte {
	if len(src) < 8 || pitch <= 0 {
		return src
	}
	const frameBytes = 4 // 16-bit stereo
	srcFrames := len(src) / frameBytes
	if srcFrames <= 1 {
		return src
	}

	outFrames := int(math.Round(float64(srcFrames) / pitch))
	if outFrames < 1 {
		outFrames = 1
	}
	dst := make([]byte, outFrames*frameBytes)

	readSample := func(frame, ch int) float64 {
		if frame < 0 {
			frame = 0
		}
		if frame >= srcFrames {
			frame = srcFrames - 1
		}
		off := frame*frameBytes + ch*2
		s := int16(binary.LittleEndian.Uint16(src[off:]))
		return float64(s) / 32767.0
	}

	for i := 0; i < outFrames; i++ {
		srcPos := float64(i) * pitch
		i0 := int(math.Floor(srcPos))
		i1 := i0 + 1
		t := srcPos - float64(i0)
		if i1 >= srcFrames {
			i1 = srcFrames - 1
		}
		for ch := 0; ch < 2; ch++ {
			s0 := readSample(i0, ch)
			s1 := readSample(i1, ch)
			v := s0 + (s1-s0)*t
			if v < -1 {
				v = -1
			} else if v > 1 {
				v = 1
			}
			off := i*frameBytes + ch*2
			binary.LittleEndian.PutUint16(dst[off:], uint16(int16(v*32767.0)))
		}
	}
	return dst
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
