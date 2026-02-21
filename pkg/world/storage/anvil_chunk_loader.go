package storage

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/tick"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

var (
	ErrMissingLevelData = errors.New("chunk nbt is missing Level data")
	ErrMissingSections  = errors.New("chunk nbt is missing Sections data")
)

type anvilChunkLoaderPending struct {
	chunkCoordinate chunk.CoordIntPair
	nbtTags         *nbt.CompoundTag
}

// AnvilChunkLoader translates core queue + IO behavior of net.minecraft.src.AnvilChunkLoader.
type AnvilChunkLoader struct {
	chunksToRemove []anvilChunkLoaderPending
	pendingCoords  map[int64]struct{}
	syncLock       sync.Mutex

	chunkSaveLocation string
}

func NewAnvilChunkLoader(chunkSaveLocation string) *AnvilChunkLoader {
	return &AnvilChunkLoader{
		chunksToRemove:    make([]anvilChunkLoaderPending, 0),
		pendingCoords:     make(map[int64]struct{}),
		chunkSaveLocation: chunkSaveLocation,
	}
}

func coordKey(coord chunk.CoordIntPair) int64 {
	return chunk.ChunkXZToInt(coord.ChunkXPos, coord.ChunkZPos)
}

// LoadChunk translates AnvilChunkLoader#loadChunk.
func (l *AnvilChunkLoader) LoadChunk(hasNoSky bool, chunkX, chunkZ int32) (*chunk.Column, []TileTickRecord, error) {
	var root *nbt.CompoundTag
	coord := chunk.NewCoordIntPair(chunkX, chunkZ)

	l.syncLock.Lock()
	if _, ok := l.pendingCoords[coordKey(coord)]; ok {
		for i := 0; i < len(l.chunksToRemove); i++ {
			if l.chunksToRemove[i].chunkCoordinate.Equals(coord) {
				root = l.chunksToRemove[i].nbtTags
				break
			}
		}
	}
	l.syncLock.Unlock()

	if root == nil {
		in, err := GetChunkInputStream(l.chunkSaveLocation, int(chunkX), int(chunkZ))
		if err != nil {
			return nil, nil, err
		}
		if in == nil {
			return nil, nil, nil
		}
		defer in.Close()

		root, err = nbt.Read(in)
		if err != nil {
			return nil, nil, err
		}
	}

	return l.checkedReadChunkFromNBT(hasNoSky, chunkX, chunkZ, root)
}

// LoadRuntimeChunk loads an anvil chunk and converts it to runtime Chunk state.
func (l *AnvilChunkLoader) LoadRuntimeChunk(world chunk.WorldBridge, hasNoSky bool, chunkX, chunkZ int32) (*chunk.Chunk, []TileTickRecord, error) {
	col, ticks, err := l.LoadChunk(hasNoSky, chunkX, chunkZ)
	if err != nil || col == nil {
		return nil, ticks, err
	}
	return chunk.NewChunkFromColumn(world, col), ticks, nil
}

func (l *AnvilChunkLoader) checkedReadChunkFromNBT(hasNoSky bool, chunkX, chunkZ int32, root *nbt.CompoundTag) (*chunk.Column, []TileTickRecord, error) {
	if root == nil || !root.HasKey("Level") {
		return nil, nil, fmt.Errorf("%w at %d,%d", ErrMissingLevelData, chunkX, chunkZ)
	}

	level, ok := root.GetTag("Level").(*nbt.CompoundTag)
	if !ok {
		return nil, nil, fmt.Errorf("%w at %d,%d", ErrMissingLevelData, chunkX, chunkZ)
	}
	if !level.HasKey("Sections") {
		return nil, nil, fmt.Errorf("%w at %d,%d", ErrMissingSections, chunkX, chunkZ)
	}

	col, tileTicks := DecodeChunkLevelNBT(level, hasNoSky)

	if col.XPos != chunkX || col.ZPos != chunkZ {
		// Translation target: AnvilChunkLoader#checkedReadChunkFromNBT relocation path.
		// Note: MCP source writes xPos/zPos to the root tag (not Level), which appears to be a bug.
		root.SetInteger("xPos", chunkX)
		root.SetInteger("zPos", chunkZ)
		col, tileTicks = DecodeChunkLevelNBT(level, hasNoSky)
	}

	return col, tileTicks, nil
}

// SaveChunk enqueues chunk NBT for async-like IO, following addChunkToPending semantics.
func (l *AnvilChunkLoader) SaveChunk(col *chunk.Column, totalWorldTime int64, hasNoSky bool, pending []*tick.NextTickListEntry) error {
	if col == nil {
		return errors.New("nil chunk column")
	}

	root := nbt.NewCompoundTag("")
	level := EncodeChunkLevelNBT(col, totalWorldTime, hasNoSky, pending)
	root.SetTag("Level", level)
	l.addChunkToPending(chunk.NewCoordIntPair(col.XPos, col.ZPos), root)
	return nil
}

// SaveRuntimeChunk converts runtime chunk state to storage column state and enqueues it.
func (l *AnvilChunkLoader) SaveRuntimeChunk(ch *chunk.Chunk, totalWorldTime int64, hasNoSky bool, pending []*tick.NextTickListEntry) error {
	if ch == nil {
		return errors.New("nil runtime chunk")
	}
	return l.SaveChunk(ch.ToColumn(), totalWorldTime, hasNoSky, pending)
}

func (l *AnvilChunkLoader) addChunkToPending(coord chunk.CoordIntPair, root *nbt.CompoundTag) {
	l.syncLock.Lock()
	defer l.syncLock.Unlock()

	key := coordKey(coord)
	if _, exists := l.pendingCoords[key]; exists {
		for i := 0; i < len(l.chunksToRemove); i++ {
			if l.chunksToRemove[i].chunkCoordinate.Equals(coord) {
				l.chunksToRemove[i] = anvilChunkLoaderPending{
					chunkCoordinate: coord,
					nbtTags:         root,
				}
				return
			}
		}
	}

	l.chunksToRemove = append(l.chunksToRemove, anvilChunkLoaderPending{
		chunkCoordinate: coord,
		nbtTags:         root,
	})
	l.pendingCoords[key] = struct{}{}
}

// WriteNextIO translates AnvilChunkLoader#writeNextIO.
func (l *AnvilChunkLoader) WriteNextIO() bool {
	var pending *anvilChunkLoaderPending

	l.syncLock.Lock()
	if len(l.chunksToRemove) == 0 {
		l.syncLock.Unlock()
		return false
	}
	first := l.chunksToRemove[0]
	l.chunksToRemove = l.chunksToRemove[1:]
	delete(l.pendingCoords, coordKey(first.chunkCoordinate))
	l.syncLock.Unlock()

	pending = &first
	if pending != nil {
		_ = l.writeChunkNBTTags(pending)
	}
	return true
}

func (l *AnvilChunkLoader) writeChunkNBTTags(pending *anvilChunkLoaderPending) error {
	out, err := GetChunkOutputStream(
		l.chunkSaveLocation,
		int(pending.chunkCoordinate.ChunkXPos),
		int(pending.chunkCoordinate.ChunkZPos),
	)
	if err != nil {
		return err
	}
	if out == nil {
		return errors.New("region output stream is nil")
	}

	if err := nbt.Write(pending.nbtTags, out); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// SaveExtraData drains all pending writes.
func (l *AnvilChunkLoader) SaveExtraData() {
	for l.WriteNextIO() {
	}
}

// ChunkTick is intentionally empty (matches MCP behavior).
func (l *AnvilChunkLoader) ChunkTick() {}

// PendingCount is test helper.
func (l *AnvilChunkLoader) PendingCount() int {
	l.syncLock.Lock()
	defer l.syncLock.Unlock()
	return len(l.chunksToRemove)
}
