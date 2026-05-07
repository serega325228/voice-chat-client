package processingpipeline

import (
	"context"
	"selfcord/internal/lib/audio"
)

const defaultChannelBufferSize = 8

type Worker func(ctx context.Context, source audio.ChannelSampleSource, sink audio.ChannelSampleSink)

func New(
	ctx context.Context,
	workers ...Worker,
) (audio.ChannelSampleSink, audio.ChannelSampleSource) {
	source, sink := audio.NewSampleChannel(defaultChannelBufferSize)
	return sink, Chain(ctx, source, workers...)
}

func Chain(
	ctx context.Context,
	source audio.ChannelSampleSource,
	workers ...Worker,
) audio.ChannelSampleSource {
	current := source

	for _, worker := range workers {
		nextSource, nextSink := audio.NewSampleChannel(defaultChannelBufferSize)
		go worker(ctx, current, nextSink)
		current = nextSource
	}

	return current
}
