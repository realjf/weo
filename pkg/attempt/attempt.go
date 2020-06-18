package attempt

import "time"

type Strategy struct {
	Total time.Duration
	Delay time.Duration
	Min int
}

type Attempt struct {
	strategy Strategy
	last     time.Time
	end      time.Time
	force    bool
	count    int
}

func (s Strategy) Run(f func() error) error {
	return s.RunWithValidator(f, func(error) bool { return true })
}

func (s Strategy) RunWithValidator(f func() error, retry func(error) bool) error {
	var err error
	for a := s.Start(); a.Next(); {
		err = f()
		if err == nil || !retry(err) {
			break
		}
	}
	return err
}

func (s Strategy) Start() *Attempt {
	now := time.Now()
	return &Attempt{
		strategy: s,
		last:     now,
		end:      now.Add(s.Total),
		force:    true,
	}
}

func (a *Attempt) Next() bool {
	now := time.Now()
	sleep := a.nextSleep(now)
	if !a.force && !now.Add(sleep).Before(a.end) && a.strategy.Min <= a.count {
		return false
	}
	a.force = false
	if sleep > 0 && a.count > 0 {
		time.Sleep(sleep)
		now = time.Now()
	}
	a.count++
	a.last = now
	return true
}

func (a *Attempt) nextSleep(now time.Time) time.Duration {
	sleep := a.strategy.Delay - now.Sub(a.last)
	if sleep < 0 {
		return 0
	}
	return sleep
}

func (a *Attempt) HasNext() bool {
	if a.force || a.strategy.Min > a.count {
		return true
	}
	now := time.Now()
	if now.Add(a.nextSleep(now)).Before(a.end) {
		a.force = true
		return true
	}
	return false
}
