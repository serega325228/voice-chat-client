package audio

import (
	"context"
	"sync"
	"unsafe"
)

const BytesPerSample = 4

type Samples []float32

type SampleSink interface {
	WriteSamples(samples Samples)
}

type SampleSource interface {
	ReadSamples(sampleCount int) Samples
}

type SamplePipeline interface {
	SampleSink
	SampleSource
}

// ChannelSampleSink/Source are non-blocking adapters for wiring services
// together via channels without stalling the audio callback.
type ChannelSampleSink chan<- Samples

func (s ChannelSampleSink) WriteSamples(samples Samples) {
	select {
	case s <- samples:
	default:
	}
}

type ChannelSampleSource <-chan Samples

func NewSampleChannel(bufferSize int) (ChannelSampleSource, ChannelSampleSink) {
	channel := make(chan Samples, bufferSize)
	return ChannelSampleSource(channel), ChannelSampleSink(channel)
}

// BufferedSampleSource preserves unread tails between ReadSamples calls.
// It is intended for pull-based consumers such as the audio playback callback.
type BufferedSampleSource struct {
	source ChannelSampleSource

	mu      sync.Mutex
	pending Samples
	closed  bool
}

func NewBufferedSampleSource(source ChannelSampleSource) *BufferedSampleSource {
	return &BufferedSampleSource{source: source}
}

func (s ChannelSampleSource) ReadSamples(sampleCount int) Samples {
	select {
	case samples := <-s:
		if len(samples) > sampleCount {
			return samples[:sampleCount]
		}
		return samples
	default:
		return nil
	}
}

func (s *BufferedSampleSource) ReadSamples(sampleCount int) Samples {
	if s == nil || sampleCount <= 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for len(s.pending) < sampleCount && !s.closed {
		select {
		case samples, ok := <-s.source:
			if !ok {
				s.closed = true
				break
			}
			if len(samples) == 0 {
				continue
			}
			s.pending = append(s.pending, samples...)
		default:
			goto drain
		}
	}

drain:
	if len(s.pending) == 0 {
		return nil
	}

	if sampleCount > len(s.pending) {
		sampleCount = len(s.pending)
	}

	out := append(Samples(nil), s.pending[:sampleCount]...)
	s.pending = s.pending[sampleCount:]

	if len(s.pending) == 0 {
		s.pending = nil
	}

	return out
}

func WriteSamplesContext(ctx context.Context, sink ChannelSampleSink, samples Samples) bool {
	select {
	case <-ctx.Done():
		return false
	case sink <- samples:
		return true
	}
}

func BytesToSamples(input []byte) Samples {
	if len(input) == 0 {
		return nil
	}

	sampleCount := len(input) / BytesPerSample
	rawSamples := unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(input))), sampleCount)

	samples := make(Samples, sampleCount)
	copy(samples, rawSamples)

	return samples
}

func WriteSamplesToBytes(output []byte, samples Samples) {
	if len(output) == 0 {
		return
	}

	clear(output)

	if len(samples) == 0 {
		return
	}

	sampleCount := len(output) / BytesPerSample
	outputSamples := unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(output))), sampleCount)
	copy(outputSamples, samples)
}
