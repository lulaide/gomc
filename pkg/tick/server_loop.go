package tick

import "time"

const (
	serverTickMillis      = int64(50)
	maxCatchupMillis      = int64(2000)
	warningCooldownMillis = int64(15000)
)

// ServerLoop translates timing behavior from net.minecraft.server.MinecraftServer#run().
type ServerLoop struct {
	Running         bool
	ServerIsRunning bool
	TimeOfLastWarn  int64

	StartServer         func() bool
	Tick                func()
	AreAllPlayersAsleep func() bool
	LogWarning          func(msg string)

	NowMillis func() int64
	Sleep     func(d time.Duration)
}

func (l *ServerLoop) nowMillis() int64 {
	if l.NowMillis != nil {
		return l.NowMillis()
	}
	return time.Now().UnixMilli()
}

func (l *ServerLoop) sleep(d time.Duration) {
	if l.Sleep != nil {
		l.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (l *ServerLoop) warn(msg string) {
	if l.LogWarning != nil {
		l.LogWarning(msg)
	}
}

func (l *ServerLoop) startServer() bool {
	if l.StartServer == nil {
		return true
	}
	return l.StartServer()
}

func (l *ServerLoop) tick() {
	if l.Tick != nil {
		l.Tick()
	}
}

func (l *ServerLoop) areAllPlayersAsleep() bool {
	if l.AreAllPlayersAsleep != nil {
		return l.AreAllPlayersAsleep()
	}
	return false
}

// Run executes the server loop until Running becomes false.
//
// Translation target:
// - net.minecraft.server.MinecraftServer#run()
func (l *ServerLoop) Run() {
	if !l.startServer() {
		return
	}

	lastMillis := l.nowMillis()
	accumulated := int64(0)
	if !l.Running {
		l.Running = true
	}

	for l.Running {
		l.ServerIsRunning = true

		now := l.nowMillis()
		delta := now - lastMillis

		if delta > maxCatchupMillis && lastMillis-l.TimeOfLastWarn >= warningCooldownMillis {
			l.warn("Can't keep up! Did the system time change, or is the server overloaded?")
			delta = maxCatchupMillis
			l.TimeOfLastWarn = lastMillis
		}

		if delta < 0 {
			l.warn("Time ran backwards! Did the system time change?")
			delta = 0
		}

		accumulated += delta
		lastMillis = now

		if l.areAllPlayersAsleep() {
			l.tick()
			accumulated = 0
		} else {
			for accumulated > serverTickMillis {
				accumulated -= serverTickMillis
				l.tick()
			}
		}

		l.sleep(time.Millisecond)
	}
}

// TickPipeline defines fixed per-tick stage ordering.
type TickPipeline interface {
	TickWeather()
	TickTime()
	TickMobSpawning()
	TickBlockUpdates()
	TickScheduledUpdates()
	TickEntities()
	TickPlayerInput()
}

// RunPipelineTick executes one 20TPS logic tick in fixed stage order.
func RunPipelineTick(p TickPipeline) {
	if p == nil {
		return
	}

	p.TickWeather()
	p.TickTime()
	p.TickMobSpawning()
	p.TickBlockUpdates()
	p.TickScheduledUpdates()
	p.TickEntities()
	p.TickPlayerInput()
}
