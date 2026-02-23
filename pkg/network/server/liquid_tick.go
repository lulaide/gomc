package server

import (
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const (
	blockIDFlowingWater = 8
	blockIDStillWater   = 9
	waterTickRate       = 5
)

const (
	blockIDSignPost = 63
	blockIDDoorWood = 64
	blockIDLadder   = 65
	blockIDDoorIron = 71
	blockIDReed     = 83
	blockIDPortal   = 90
)

var waterFlowDirections = [4][2]int{
	{-1, 0}, // west
	{1, 0},  // east
	{0, -1}, // north
	{0, 1},  // south
}

type waterBlockPos struct {
	x int
	y int
	z int
}

type waterPlacement struct {
	blockID int
	meta    int
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

func waterPlacementStrength(blockID, meta int) int {
	if blockID == 0 {
		return 10_000
	}
	if !isWaterBlockID(blockID) {
		return 9_000
	}
	if blockID == blockIDStillWater && normalizedWaterLevel(meta) == 0 && meta < 8 {
		return 0
	}
	if meta < 8 {
		return 10 + meta
	}
	return 18
}

func (s *StatusServer) proposeWaterPlacement(changes map[waterBlockPos]waterPlacement, x, y, z, blockID, meta int) {
	if y < 0 || y >= 256 {
		return
	}
	if blockID != 0 {
		if meta < 0 {
			meta = 0
		}
		if meta > 8 {
			meta = 8
		}
	}

	pos := waterBlockPos{x: x, y: y, z: z}
	cur, ok := changes[pos]
	next := waterPlacement{blockID: blockID, meta: meta}
	if !ok || waterPlacementStrength(next.blockID, next.meta) < waterPlacementStrength(cur.blockID, cur.meta) {
		changes[pos] = next
	}
}

func (s *StatusServer) waterFlowDecay(x, y, z int) int {
	id, meta := s.world.getBlock(x, y, z)
	if !isWaterBlockID(id) {
		return -1
	}
	return meta
}

func (s *StatusServer) effectiveWaterFlowDecay(x, y, z int) int {
	decay := s.waterFlowDecay(x, y, z)
	if decay < 0 {
		return -1
	}
	if decay >= 8 {
		decay = 0
	}
	return decay
}

func (s *StatusServer) waterBlockBlocksFlow(x, y, z int) bool {
	id, _ := s.world.getBlock(x, y, z)
	switch id {
	case 0:
		return false
	case blockIDDoorWood, blockIDDoorIron, blockIDSignPost, blockIDLadder, blockIDReed:
		return true
	case blockIDPortal:
		return true
	}
	return block.BlocksMovement(id)
}

func (s *StatusServer) waterCanDisplaceBlock(x, y, z int) bool {
	id, _ := s.world.getBlock(x, y, z)
	if isWaterBlockID(id) {
		return false
	}
	return !s.waterBlockBlocksFlow(x, y, z)
}

func (s *StatusServer) getSmallestWaterFlowDecay(x, y, z, smallest int, adjacentSources *int) int {
	decay := s.waterFlowDecay(x, y, z)
	if decay < 0 {
		return smallest
	}
	if decay == 0 {
		*adjacentSources = *adjacentSources + 1
	}
	if decay >= 8 {
		decay = 0
	}
	if smallest < 0 || decay < smallest {
		return decay
	}
	return smallest
}

func (s *StatusServer) calculateWaterFlowCost(x, y, z, accumulatedCost, previousDirection int) int {
	best := 1000
	for dir := 0; dir < 4; dir++ {
		if (dir == 0 && previousDirection == 1) ||
			(dir == 1 && previousDirection == 0) ||
			(dir == 2 && previousDirection == 3) ||
			(dir == 3 && previousDirection == 2) {
			continue
		}
		delta := waterFlowDirections[dir]
		nx := x + delta[0]
		nz := z + delta[1]
		neighborID, neighborMeta := s.world.getBlock(nx, y, nz)
		if s.waterBlockBlocksFlow(nx, y, nz) || (isWaterBlockID(neighborID) && neighborMeta == 0) {
			continue
		}
		if !s.waterBlockBlocksFlow(nx, y-1, nz) {
			return accumulatedCost
		}
		if accumulatedCost >= 4 {
			continue
		}
		cost := s.calculateWaterFlowCost(nx, y, nz, accumulatedCost+1, dir)
		if cost < best {
			best = cost
		}
	}
	return best
}

func (s *StatusServer) optimalWaterFlowDirections(x, y, z int) [4]bool {
	flowCost := [4]int{1000, 1000, 1000, 1000}
	for dir := 0; dir < 4; dir++ {
		delta := waterFlowDirections[dir]
		nx := x + delta[0]
		nz := z + delta[1]
		neighborID, neighborMeta := s.world.getBlock(nx, y, nz)
		if s.waterBlockBlocksFlow(nx, y, nz) || (isWaterBlockID(neighborID) && neighborMeta == 0) {
			continue
		}
		if s.waterBlockBlocksFlow(nx, y-1, nz) {
			flowCost[dir] = s.calculateWaterFlowCost(nx, y, nz, 1, dir)
		} else {
			flowCost[dir] = 0
		}
	}
	best := flowCost[0]
	for i := 1; i < 4; i++ {
		if flowCost[i] < best {
			best = flowCost[i]
		}
	}
	var out [4]bool
	for i := 0; i < 4; i++ {
		out[i] = flowCost[i] == best
	}
	return out
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

	positions := make([]waterBlockPos, 0, 256)
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
					positions = append(positions, waterBlockPos{x: worldX, y: y, z: worldZ})
				}
			}
		}
	}

	changes := make(map[waterBlockPos]waterPlacement, len(positions)*2)
	for _, pos := range positions {
		flowDecay := s.waterFlowDecay(pos.x, pos.y, pos.z)
		if flowDecay < 0 {
			continue
		}

		if flowDecay > 0 {
			smallest := -100
			adjacentSources := 0
			smallest = s.getSmallestWaterFlowDecay(pos.x-1, pos.y, pos.z, smallest, &adjacentSources)
			smallest = s.getSmallestWaterFlowDecay(pos.x+1, pos.y, pos.z, smallest, &adjacentSources)
			smallest = s.getSmallestWaterFlowDecay(pos.x, pos.y, pos.z-1, smallest, &adjacentSources)
			smallest = s.getSmallestWaterFlowDecay(pos.x, pos.y, pos.z+1, smallest, &adjacentSources)

			newDecay := smallest + 1
			if newDecay >= 8 || smallest < 0 {
				newDecay = -1
			}

			aboveDecay := s.waterFlowDecay(pos.x, pos.y+1, pos.z)
			if aboveDecay >= 0 {
				if aboveDecay >= 8 {
					newDecay = aboveDecay
				} else {
					newDecay = aboveDecay + 8
				}
			}

			if adjacentSources >= 2 {
				belowID, belowMeta := s.world.getBlock(pos.x, pos.y-1, pos.z)
				if block.BlocksMovement(belowID) || (isWaterBlockID(belowID) && belowMeta == 0) {
					newDecay = 0
				}
			}

			if newDecay == flowDecay {
				// Translation reference:
				// - net.minecraft.src.BlockFlowing#updateFlow(...)
				// Stable flowing block becomes still water with same metadata.
				s.proposeWaterPlacement(changes, pos.x, pos.y, pos.z, blockIDStillWater, flowDecay)
			} else {
				flowDecay = newDecay
				if flowDecay < 0 {
					s.proposeWaterPlacement(changes, pos.x, pos.y, pos.z, 0, 0)
				} else {
					s.proposeWaterPlacement(changes, pos.x, pos.y, pos.z, blockIDFlowingWater, flowDecay)
				}
			}
		} else {
			s.proposeWaterPlacement(changes, pos.x, pos.y, pos.z, blockIDStillWater, flowDecay)
		}

		if flowDecay < 0 {
			continue
		}

		if s.waterCanDisplaceBlock(pos.x, pos.y-1, pos.z) {
			// Falling water keeps metadata in [8..15] wire format.
			downMeta := flowDecay
			if downMeta < 8 {
				downMeta += 8
			}
			s.proposeWaterPlacement(changes, pos.x, pos.y-1, pos.z, blockIDFlowingWater, downMeta)
			continue
		}

		if flowDecay < 0 || (flowDecay != 0 && !s.waterBlockBlocksFlow(pos.x, pos.y-1, pos.z)) {
			continue
		}

		nextDecay := flowDecay + 1
		if flowDecay >= 8 {
			nextDecay = 1
		}
		if nextDecay >= 8 {
			continue
		}

		optimal := s.optimalWaterFlowDirections(pos.x, pos.y, pos.z)
		for dir, ok := range optimal {
			if !ok {
				continue
			}
			delta := waterFlowDirections[dir]
			nx := pos.x + delta[0]
			nz := pos.z + delta[1]
			if !s.waterCanDisplaceBlock(nx, pos.y, nz) {
				continue
			}
			s.proposeWaterPlacement(changes, nx, pos.y, nz, blockIDFlowingWater, nextDecay)
		}
	}

	for pos, update := range changes {
		oldID, oldMeta := s.world.getBlock(pos.x, pos.y, pos.z)
		if update.blockID == 0 {
			if oldID == 0 {
				continue
			}
			if s.world.setBlock(pos.x, pos.y, pos.z, 0, 0) {
				s.broadcastBlockChange(int32(pos.x), int32(pos.y), int32(pos.z), 0, 0)
			}
			continue
		}

		if !isWaterBlockID(oldID) && !s.waterCanDisplaceBlock(pos.x, pos.y, pos.z) {
			continue
		}
		// Keep source blocks stable unless replaced by another source-level update.
		if oldID == blockIDStillWater && normalizedWaterLevel(oldMeta) == 0 && oldMeta < 8 {
			if !(update.blockID == blockIDStillWater && normalizedWaterLevel(update.meta) == 0 && update.meta < 8) {
				continue
			}
		}

		if s.world.setBlock(pos.x, pos.y, pos.z, update.blockID, update.meta) {
			s.broadcastBlockChange(int32(pos.x), int32(pos.y), int32(pos.z), int32(update.blockID), int32(update.meta))
		}
	}
}
