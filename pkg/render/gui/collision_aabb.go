//go:build cgo

package gui

type axisAlignedBB struct {
	minX float64
	minY float64
	minZ float64
	maxX float64
	maxY float64
	maxZ float64
}

func playerBoundingBox(px, py, pz float64) axisAlignedBB {
	return axisAlignedBB{
		minX: px - playerHalfWidth,
		minY: py,
		minZ: pz - playerHalfWidth,
		maxX: px + playerHalfWidth,
		maxY: py + playerHeight,
		maxZ: pz + playerHalfWidth,
	}
}

func unitBlockAABB(x, y, z int) axisAlignedBB {
	return axisAlignedBB{
		minX: float64(x),
		minY: float64(y),
		minZ: float64(z),
		maxX: float64(x + 1),
		maxY: float64(y + 1),
		maxZ: float64(z + 1),
	}
}

func (bb axisAlignedBB) addCoord(dx, dy, dz float64) axisAlignedBB {
	out := bb
	if dx < 0 {
		out.minX += dx
	} else if dx > 0 {
		out.maxX += dx
	}
	if dy < 0 {
		out.minY += dy
	} else if dy > 0 {
		out.maxY += dy
	}
	if dz < 0 {
		out.minZ += dz
	} else if dz > 0 {
		out.maxZ += dz
	}
	return out
}

func (bb axisAlignedBB) offset(dx, dy, dz float64) axisAlignedBB {
	return axisAlignedBB{
		minX: bb.minX + dx,
		minY: bb.minY + dy,
		minZ: bb.minZ + dz,
		maxX: bb.maxX + dx,
		maxY: bb.maxY + dy,
		maxZ: bb.maxZ + dz,
	}
}

func (bb axisAlignedBB) intersects(other axisAlignedBB) bool {
	return other.maxX > bb.minX &&
		other.minX < bb.maxX &&
		other.maxY > bb.minY &&
		other.minY < bb.maxY &&
		other.maxZ > bb.minZ &&
		other.minZ < bb.maxZ
}

func (bb axisAlignedBB) calculateXOffset(other axisAlignedBB, offsetX float64) float64 {
	if other.maxY <= bb.minY || other.minY >= bb.maxY {
		return offsetX
	}
	if other.maxZ <= bb.minZ || other.minZ >= bb.maxZ {
		return offsetX
	}
	if offsetX > 0 && other.maxX <= bb.minX {
		d := bb.minX - other.maxX
		if d < offsetX {
			offsetX = d
		}
	} else if offsetX < 0 && other.minX >= bb.maxX {
		d := bb.maxX - other.minX
		if d > offsetX {
			offsetX = d
		}
	}
	return offsetX
}

func (bb axisAlignedBB) calculateYOffset(other axisAlignedBB, offsetY float64) float64 {
	if other.maxX <= bb.minX || other.minX >= bb.maxX {
		return offsetY
	}
	if other.maxZ <= bb.minZ || other.minZ >= bb.maxZ {
		return offsetY
	}
	if offsetY > 0 && other.maxY <= bb.minY {
		d := bb.minY - other.maxY
		if d < offsetY {
			offsetY = d
		}
	} else if offsetY < 0 && other.minY >= bb.maxY {
		d := bb.maxY - other.minY
		if d > offsetY {
			offsetY = d
		}
	}
	return offsetY
}

func (bb axisAlignedBB) calculateZOffset(other axisAlignedBB, offsetZ float64) float64 {
	if other.maxX <= bb.minX || other.minX >= bb.maxX {
		return offsetZ
	}
	if other.maxY <= bb.minY || other.minY >= bb.maxY {
		return offsetZ
	}
	if offsetZ > 0 && other.maxZ <= bb.minZ {
		d := bb.minZ - other.maxZ
		if d < offsetZ {
			offsetZ = d
		}
	} else if offsetZ < 0 && other.minZ >= bb.maxZ {
		d := bb.maxZ - other.minZ
		if d > offsetZ {
			offsetZ = d
		}
	}
	return offsetZ
}
