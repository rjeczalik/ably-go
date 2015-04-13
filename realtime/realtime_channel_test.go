package realtime_test

import (
	"github.com/ably/ably-go/realtime"

	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/gomega"
)

var _ = Describe("RealtimeChannel", func() {
	var (
		client  *realtime.RealtimeClient
		channel *realtime.RealtimeChannel
	)

	BeforeEach(func() {
		client = realtime.NewRealtimeClient(testApp.Params)
		channel = client.RealtimeChannel("test")
	})

	AfterEach(func() {
		client.Close()
	})

	Context("When the connection is ready", func() {
		XIt("publish messages", func() {
			err := channel.Publish("hello", "world")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
