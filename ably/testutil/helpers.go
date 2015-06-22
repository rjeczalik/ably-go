package testutil

import "github.com/ably/ably-go/ably"

func mergeOpts(opts, override *ably.ClientOptions) *ably.ClientOptions {
	if override == nil {
		return opts
	}
	opts.AuthOptions.Set(&override.AuthOptions)
	if override.RestHost != "" {
		opts.RestHost = override.RestHost
	}
	if override.RealtimeHost != "" {
		opts.RealtimeHost = override.RealtimeHost
	}
	if override.Environment != "" {
		opts.Environment = override.Environment
	}
	if override.ClientID != "" {
		opts.ClientID = override.ClientID
	}
	if override.Protocol != "" {
		opts.Protocol = override.Protocol
	}
	if override.Recover != "" {
		opts.Recover = override.Recover
	}
	if override.NoTLS {
		opts.NoTLS = true
	}
	if override.NoConnect {
		opts.NoConnect = true
	}
	if override.NoEcho {
		opts.NoEcho = true
	}
	if override.NoQueueing {
		opts.NoQueueing = true
	}
	if override.TimeoutConnect != 0 {
		opts.TimeoutConnect = override.TimeoutConnect
	}
	if override.TimeoutDisconnect != 0 {
		opts.TimeoutDisconnect = override.TimeoutDisconnect
	}
	if override.TimeoutSuspended != 0 {
		opts.TimeoutSuspended = override.TimeoutSuspended
	}
	if override.Dial != nil {
		opts.Dial = override.Dial
	}
	if override.Listener != nil {
		opts.Listener = override.Listener
	}
	if override.HTTPClient != nil {
		opts.HTTPClient = override.HTTPClient
	}
	return opts
}

func MergeOptions(opts ...*ably.ClientOptions) *ably.ClientOptions {
	switch len(opts) {
	case 0:
		return nil
	case 1:
		return opts[0]
	}
	mergedOpts := opts[0]
	for _, opt := range opts[1:] {
		mergedOpts = mergeOpts(mergedOpts, opt)
	}
	return mergedOpts
}
