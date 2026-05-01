package audio

import "sync"

type MuteableSampleSink struct {
	target SampleSink

	mu    sync.RWMutex
	muted bool
}

func NewMuteableSampleSink(target SampleSink) *MuteableSampleSink {
	return &MuteableSampleSink{target: target}
}

func (s *MuteableSampleSink) WriteSamples(samples Samples) {
	if s == nil || s.target == nil {
		return
	}

	s.mu.RLock()
	muted := s.muted
	s.mu.RUnlock()
	if muted {
		return
	}

	s.target.WriteSamples(samples)
}

func (s *MuteableSampleSink) SetMuted(muted bool) {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.muted = muted
	s.mu.Unlock()
}

func (s *MuteableSampleSink) IsMuted() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.muted
}
