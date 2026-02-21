package gui

import "math"

// movementDeltaFromYaw mirrors 1.6.4 local move vector math:
// EntityLivingBase.moveEntityWithHeading(strafe, forward)
// motionX += strafe*cos(yaw) - forward*sin(yaw)
// motionZ += forward*cos(yaw) + strafe*sin(yaw)
func movementDeltaFromYaw(yawDeg, forward, strafe, speed float64) (dx, dz float64) {
	yawRad := yawDeg * math.Pi / 180.0
	dx = (-math.Sin(yawRad)*forward + math.Cos(yawRad)*strafe) * speed
	dz = (math.Cos(yawRad)*forward + math.Sin(yawRad)*strafe) * speed
	return dx, dz
}

func lookDirectionFromYawPitch(yawDeg, pitchDeg float64) (dx, dy, dz float64) {
	yawRad := yawDeg * math.Pi / 180.0
	pitchRad := pitchDeg * math.Pi / 180.0
	dx = -math.Sin(yawRad) * math.Cos(pitchRad)
	dy = -math.Sin(pitchRad)
	dz = math.Cos(yawRad) * math.Cos(pitchRad)
	return dx, dy, dz
}
