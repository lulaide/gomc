package server

import (
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const (
	blockIDFlowingWater = 8
	blockIDStillWater   = 9
)

var waterFlowDirections = [][2]int{
	{1, 0},
	{-1, 0},
	{0, 1},
	{0, -1},
}

type waterBlockPos struct {
	x int
	y int
	z int
}

type waterPlacement struct {
	meta int
}

func isWaterBlockID(id int) bool {
	return id == blockIDFlowingWater || id == blockIDStillWater
}

func normalizedWaterLevel(meta int) int {
	if meta >= 8 {
		return 0
	}
	return meta & 7
}

func (s *StatusServer) canWaterFlowInto(blockID int) bool {
	if blockID == 0 {
		return true
	}
	if block.IsLiquid(blockID) {
		return false
	}
	return !block.BlocksMovement(blockID)
}

func (s *StatusServer) proposeWaterPlacement(changes map[waterBlockPos]waterPlacement, x, y, z, meta int) {
	if y < 0 || y >= 256 {
		return
	}
	if meta < 0 {
		meta = 0
	}
	if meta > 8 {
		meta = 8
	}

	pos := waterBlockPos{x: x, y: y, z: z}
	cur, ok := changes[pos]
	if !ok || meta < cur.meta {
		changes[pos] = waterPlacement{meta: meta}
	}
}

func (s *StatusServer) activeLoadedChunksSnapshot() map[chunk.CoordIntPair]struct{} {
	s.activeMu.RLock()
	sessions := make([]*loginSession, 0, len(s.activePlayers))
	for session := range s.activePlayers {
		sessions = append(sessions, session)
	}
	s.activeMu.RUnlock()

	if len(sessions) == 0 {
		return nil
	}

	out := make(map[chunk.CoordIntPair]struct{}, len(sessions)*64)
	for _, session := range sessions {
		if session == nil {
			continue
		}
		session.stateMu.Lock()
		alive := session.playerRegistered && !session.playerDead && session.entityID != 0
		if alive {
			for pos := range session.loadedChunks {
				out[pos] = struct{}{}
			}
		}
		session.stateMu.Unlock()
	}
	return out
}

// Translation reference:
// - net.minecraft.src.BlockFlowing.updateTick(...)
// - net.minecraft.src.BlockFluid metadata levels (0..7 normal, 8..15 falling)
func (s *StatusServer) tickWaterFlow() {
	loadedChunks := s.activeLoadedChunksSnapshot()
	if len(loadedChunks) == 0 {
		return
	}

	changes := make(map[waterBlockPos]waterPlacement, 256)
	for coord := range loadedChunks {
		ch := s.world.getChunk(coord.ChunkXPos, coord.ChunkZPos)
		if ch == nil {
			continue
		}

		topY := ch.GetTopFilledSegment() + 16
		if topY < 1 {
			continue
		}
		if topY > 255 {
			topY = 255
		}

		baseX := int(coord.ChunkXPos) << 4
		baseZ := int(coord.ChunkZPos) << 4
		for localX := 0; localX < 16; localX++ {
			worldX := baseX + localX
			for localZ := 0; localZ < 16; localZ++ {
				worldZ := baseZ + localZ
				for y := 1; y <= topY; y++ {
					id := ch.GetBlockID(localX, y, localZ)
					if !isWaterBlockID(id) {
						continue
					}
					meta := ch.GetBlockMetadata(localX, y, localZ)
					level := normalizedWaterLevel(meta)

					belowID, belowMeta := s.world.getBlock(worldX, y-1, worldZ)
					if s.canWaterFlowInto(belowID) || (isWaterBlockID(belowID) && normalizedWaterLevel(belowMeta) > 0) {
						// Falling water keeps meta>=8 like vanilla wire format.
						s.proposeWaterPlacement(changes, worldX, y-1, worldZ, 8)
						continue
					}

					nextLevel := level + 1
					if nextLevel > 7 {
						continue
					}
					for _, dir := range waterFlowDirections {
						nx := worldX + dir[0]
						nz := worldZ + dir[1]
						nID, nMeta := s.world.getBlock(nx, y, nz)
						if isWaterBlockID(nID) {
							if normalizedWaterLevel(nMeta) <= nextLevel {
								continue
							}
							s.proposeWaterPlacement(changes, nx, y, nz, nextLevel)
							continue
						}
						if !s.canWaterFlowInto(nID) {
							continue
						}
						s.proposeWaterPlacement(changes, nx, y, nz, nextLevel)
					}
				}
			}
		}
	}

	for pos, update := range changes {
		oldID, oldMeta := s.world.getBlock(pos.x, pos.y, pos.z)
		if isWaterBlockID(oldID) {
			// Keep still sources stable.
			if oldID == blockIDStillWater && normalizedWaterLevel(oldMeta) == 0 && oldMeta < 8 {
				continue
			}
			// Existing stronger/equal water wins.
			if oldMeta < 8 && normalizedWaterLevel(oldMeta) <= update.meta {
				continue
			}
		} else if !s.canWaterFlowInto(oldID) {
			continue
		}

		if s.world.setBlock(pos.x, pos.y, pos.z, blockIDFlowingWater, update.meta) {
			s.broadcastBlockChange(int32(pos.x), int32(pos.y), int32(pos.z), blockIDFlowingWater, int32(update.meta))
		}
	}
}
