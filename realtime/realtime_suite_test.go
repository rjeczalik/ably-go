package realtime_test

import (
	"testing"

	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/ably/ably-go/realtime"
	"github.com/ably/ably-go/test/support"
)

func TestRealtime(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Realtime Suite")
}

var (
	testApp *support.TestApp
	client  *realtime.RealtimeClient
	channel *realtime.RealtimeChannel
)

var _ = BeforeSuite(func() {
	testApp = support.NewTestApp()
	_, err := testApp.Create()
	Expect(err).NotTo(HaveOccurred())
})

var _ = BeforeEach(func() {
	client = realtime.NewRealtimeClient(testApp.Params)
	channel = client.RealtimeChannel("test")
})

var _ = AfterEach(func() {
	client.Close()
})

var _ = AfterSuite(func() {
	_, err := testApp.Delete()
	Expect(err).NotTo(HaveOccurred())
})
