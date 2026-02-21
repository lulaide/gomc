package gen

// floorDoubleLong matches net.minecraft.src.MathHelper#floor_double_long.
func floorDoubleLong(v float64) int64 {
	n := int64(v)
	if v < float64(n) {
		return n - 1
	}
	return n
}

// floorDouble matches net.minecraft.src.MathHelper#floor_double.
func floorDouble(v float64) int {
	n := int(v)
	if v < float64(n) {
		return n - 1
	}
	return n
}
