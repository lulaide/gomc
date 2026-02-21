package block

import "sync"

const maxBlockID = 4095

// Definition mirrors Block#isAssociatedBlockID contract.
//
// Translation target:
// - net.minecraft.src.Block#isAssociatedBlockID(int)
type Definition interface {
	ID() int
	IsAssociatedBlockID(otherID int) bool
}

// TickRandomlyProvider mirrors Block#getTickRandomly().
type TickRandomlyProvider interface {
	GetTickRandomly() bool
}

// LightOpacityProvider mirrors reads from Block.lightOpacity[id].
type LightOpacityProvider interface {
	GetLightOpacity() int
}

// MaterialProvider mirrors Material#blocksMovement/#isLiquid checks used by Chunk.
type MaterialProvider interface {
	BlocksMovement() bool
	IsLiquid() bool
}

// TileEntityProvider mirrors `instanceof ITileEntityProvider` checks.
type TileEntityProvider interface {
	IsTileEntityProvider() bool
}

type baseDefinition struct {
	id int
}

func NewBaseDefinition(id int) Definition {
	return &baseDefinition{id: id}
}

func (d *baseDefinition) ID() int {
	return d.id
}

func (d *baseDefinition) IsAssociatedBlockID(otherID int) bool {
	return d.id == otherID
}

func (d *baseDefinition) GetTickRandomly() bool {
	return false
}

var (
	registryMu sync.RWMutex
	blocksList [maxBlockID + 1]Definition

	lightOpacity     [maxBlockID + 1]int
	blocksMovement   [maxBlockID + 1]bool
	isLiquid         [maxBlockID + 1]bool
	tileEntityBlocks [maxBlockID + 1]bool
)

func Register(def Definition) {
	if def == nil {
		return
	}
	id := def.ID()
	if id < 0 || id > maxBlockID {
		return
	}
	registryMu.Lock()
	blocksList[id] = def
	lightOpacity[id] = 0
	blocksMovement[id] = false
	isLiquid[id] = false
	tileEntityBlocks[id] = false

	if provider, ok := def.(LightOpacityProvider); ok {
		lightOpacity[id] = provider.GetLightOpacity()
	}
	if provider, ok := def.(MaterialProvider); ok {
		blocksMovement[id] = provider.BlocksMovement()
		isLiquid[id] = provider.IsLiquid()
	}
	if provider, ok := def.(TileEntityProvider); ok {
		tileEntityBlocks[id] = provider.IsTileEntityProvider()
	}
	registryMu.Unlock()
}

func Unregister(id int) {
	if id < 0 || id > maxBlockID {
		return
	}
	registryMu.Lock()
	blocksList[id] = nil
	lightOpacity[id] = 0
	blocksMovement[id] = false
	isLiquid[id] = false
	tileEntityBlocks[id] = false
	registryMu.Unlock()
}

func ResetRegistry() {
	registryMu.Lock()
	for i := range blocksList {
		blocksList[i] = nil
		lightOpacity[i] = 0
		blocksMovement[i] = false
		isLiquid[i] = false
		tileEntityBlocks[i] = false
	}
	registryMu.Unlock()
}

func Lookup(id int) Definition {
	if id < 0 || id > maxBlockID {
		return nil
	}
	registryMu.RLock()
	def := blocksList[id]
	registryMu.RUnlock()
	return def
}

func Exists(id int) bool {
	return Lookup(id) != nil
}

func GetTickRandomly(id int) bool {
	def := Lookup(id)
	if def == nil {
		return false
	}
	provider, ok := def.(TickRandomlyProvider)
	return ok && provider.GetTickRandomly()
}

// SetLightOpacity sets static block light opacity array value.
//
// Translation target:
// - net.minecraft.src.Block.lightOpacity[id]
func SetLightOpacity(id int, opacity int) {
	if id < 0 || id > maxBlockID {
		return
	}
	registryMu.Lock()
	lightOpacity[id] = opacity
	registryMu.Unlock()
}

func GetLightOpacity(id int) int {
	if id < 0 || id > maxBlockID {
		return 0
	}
	registryMu.RLock()
	v := lightOpacity[id]
	registryMu.RUnlock()
	return v
}

// SetMaterialProperties sets movement/liquid behavior used by precipitation checks.
func SetMaterialProperties(id int, movement bool, liquid bool) {
	if id < 0 || id > maxBlockID {
		return
	}
	registryMu.Lock()
	blocksMovement[id] = movement
	isLiquid[id] = liquid
	registryMu.Unlock()
}

func BlocksMovement(id int) bool {
	if id < 0 || id > maxBlockID {
		return false
	}
	registryMu.RLock()
	v := blocksMovement[id]
	registryMu.RUnlock()
	return v
}

func IsLiquid(id int) bool {
	if id < 0 || id > maxBlockID {
		return false
	}
	registryMu.RLock()
	v := isLiquid[id]
	registryMu.RUnlock()
	return v
}

func SetTileEntityProvider(id int, enabled bool) {
	if id < 0 || id > maxBlockID {
		return
	}
	registryMu.Lock()
	tileEntityBlocks[id] = enabled
	registryMu.Unlock()
}

func IsTileEntityProvider(id int) bool {
	if id < 0 || id > maxBlockID {
		return false
	}
	registryMu.RLock()
	v := tileEntityBlocks[id]
	registryMu.RUnlock()
	return v
}

// IsAssociatedBlockID translates net.minecraft.src.Block#isAssociatedBlockID(int, int).
func IsAssociatedBlockID(a, b int) bool {
	if a == b {
		return true
	}
	if a == 0 || b == 0 {
		return false
	}
	if a < 0 || a > maxBlockID || b < 0 || b > maxBlockID {
		return false
	}

	registryMu.RLock()
	ba := blocksList[a]
	bb := blocksList[b]
	registryMu.RUnlock()

	if ba == nil || bb == nil {
		return false
	}
	return ba.IsAssociatedBlockID(b)
}
