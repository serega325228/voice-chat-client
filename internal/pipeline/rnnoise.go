package processingpipeline

/*
#cgo LDFLAGS: -lrnnoise
#include <rnnoise.h>
*/
import "C"
import (
	"unsafe"

	"context"
	"selfcord/internal/lib/audio"
)

const rnnoiseFrameSize = 480

func RNNoiseWorker(ctx context.Context, source audio.ChannelSampleSource, sink audio.ChannelSampleSink) {
	defer close(sink)

	state := C.rnnoise_create(nil)
	if state == nil {
		forwardSamples(ctx, source, sink)
		return
	}
	defer C.rnnoise_destroy(state)

	frame := make(audio.Samples, 0, rnnoiseFrameSize)

	for {
		select {
		case <-ctx.Done():
			return
		case samples, ok := <-source:
			if !ok {
				if len(frame) > 0 {
					tail := append(audio.Samples(nil), frame...)
					audio.WriteSamplesContext(ctx, sink, tail)
				}
				return
			}

			for len(samples) > 0 {
				missing := rnnoiseFrameSize - len(frame)
				if missing > len(samples) {
					missing = len(samples)
				}

				frame = append(frame, samples[:missing]...)
				samples = samples[missing:]

				if len(frame) < rnnoiseFrameSize {
					continue
				}

				processRNNoiseFrame(state, frame)

				processed := append(audio.Samples(nil), frame...)
				if !audio.WriteSamplesContext(ctx, sink, processed) {
					return
				}

				frame = frame[:0]
			}
		}
	}
}

func forwardSamples(ctx context.Context, source audio.ChannelSampleSource, sink audio.ChannelSampleSink) {
	for {
		select {
		case <-ctx.Done():
			return
		case samples, ok := <-source:
			if !ok {
				return
			}
			if !audio.WriteSamplesContext(ctx, sink, samples) {
				return
			}
		}
	}
}

func processRNNoiseFrame(state *C.DenoiseState, frame audio.Samples) {
	ptr := (*C.float)(unsafe.Pointer(&frame[0]))
	C.rnnoise_process_frame(state, ptr, ptr)
}
