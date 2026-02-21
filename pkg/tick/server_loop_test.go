package tick

import (
	"testing"
	"time"
)

type loopClock struct {
	values []int64
	idx    int
}

func (c *loopClock) now() int64 {
	if c.idx >= len(c.values) {
		return c.values[len(c.values)-1]
	}
	v := c.values[c.idx]
	c.idx++
	return v
}

func TestServerLoopCatchupTicks(t *testing.T) {
	clock := &loopClock{values: []int64{1000, 1120}}
	ticks := 0

	var loop *ServerLoop
	loop = &ServerLoop{
		Running: true,
		StartServer: func() bool {
			return true
		},
		NowMillis: clock.now,
		Tick: func() {
			ticks++
		},
		Sleep: func(d time.Duration) {
			loop.Running = false
		},
	}

	loop.Run()
	if ticks != 2 {
		t.Fatalf("tick count mismatch: got=%d want=2", ticks)
	}
}

func TestServerLoopAllPlayersAsleepTicksOnce(t *testing.T) {
	clock := &loopClock{values: []int64{2000, 2200}}
	ticks := 0

	var loop *ServerLoop
	loop = &ServerLoop{
		Running: true,
		StartServer: func() bool {
			return true
		},
		NowMillis: clock.now,
		AreAllPlayersAsleep: func() bool {
			return true
		},
		Tick: func() {
			ticks++
		},
		Sleep: func(d time.Duration) {
			loop.Running = false
		},
	}

	loop.Run()
	if ticks != 1 {
		t.Fatalf("sleeping tick count mismatch: got=%d want=1", ticks)
	}
}

func TestServerLoopWarnsOnBackwardTime(t *testing.T) {
	clock := &loopClock{values: []int64{1000, 900}}
	ticks := 0
	warnings := 0

	var loop *ServerLoop
	loop = &ServerLoop{
		Running: true,
		StartServer: func() bool {
			return true
		},
		NowMillis: clock.now,
		Tick: func() {
			ticks++
		},
		LogWarning: func(msg string) {
			warnings++
		},
		Sleep: func(d time.Duration) {
			loop.Running = false
		},
	}

	loop.Run()
	if warnings != 1 {
		t.Fatalf("warning count mismatch: got=%d want=1", warnings)
	}
	if ticks != 0 {
		t.Fatalf("tick count mismatch on backward time: got=%d want=0", ticks)
	}
}

func TestServerLoopOverloadClampAndWarningCooldown(t *testing.T) {
	clock := &loopClock{values: []int64{20000, 24010}}
	ticks := 0
	warnings := 0

	var loop *ServerLoop
	loop = &ServerLoop{
		Running: true,
		StartServer: func() bool {
			return true
		},
		NowMillis:      clock.now,
		TimeOfLastWarn: 0,
		Tick: func() {
			ticks++
		},
		LogWarning: func(msg string) {
			warnings++
		},
		Sleep: func(d time.Duration) {
			loop.Running = false
		},
	}

	loop.Run()

	if warnings != 1 {
		t.Fatalf("warning count mismatch: got=%d want=1", warnings)
	}
	if ticks != 39 {
		t.Fatalf("tick count mismatch with 2000ms clamp: got=%d want=39", ticks)
	}
	if loop.TimeOfLastWarn != 20000 {
		t.Fatalf("TimeOfLastWarn mismatch: got=%d want=20000", loop.TimeOfLastWarn)
	}
}

type orderPipeline struct {
	order []string
}

func (p *orderPipeline) TickWeather()          { p.order = append(p.order, "weather") }
func (p *orderPipeline) TickTime()             { p.order = append(p.order, "time") }
func (p *orderPipeline) TickMobSpawning()      { p.order = append(p.order, "mob") }
func (p *orderPipeline) TickBlockUpdates()     { p.order = append(p.order, "block") }
func (p *orderPipeline) TickScheduledUpdates() { p.order = append(p.order, "scheduled") }
func (p *orderPipeline) TickEntities()         { p.order = append(p.order, "entity") }
func (p *orderPipeline) TickPlayerInput()      { p.order = append(p.order, "player") }

func TestRunPipelineTickOrder(t *testing.T) {
	p := &orderPipeline{}
	RunPipelineTick(p)

	want := []string{"weather", "time", "mob", "block", "scheduled", "entity", "player"}
	if len(p.order) != len(want) {
		t.Fatalf("stage count mismatch: got=%d want=%d", len(p.order), len(want))
	}
	for i := range want {
		if p.order[i] != want[i] {
			t.Fatalf("stage order mismatch at %d: got=%q want=%q", i, p.order[i], want[i])
		}
	}
}
