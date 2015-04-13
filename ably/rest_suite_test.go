package ably_test

import (
	"github.com/ably/ably-go/ably"
	"github.com/ably/ably-go/test/support"

	"testing"

	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/gomega"
)

func TestRest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rest Suite")
}

var (
	testApp *support.TestApp
	client  *ably.RestClient
	channel *ably.RestChannel
)

var _ = BeforeSuite(func() {
	testApp = support.NewTestApp()
	_, err := testApp.Create()
	Expect(err).NotTo(HaveOccurred())
})

var _ = BeforeEach(func() {
	client = ably.NewRestClient(testApp.Params)
	channel = client.RestChannel("test")
})

var _ = AfterSuite(func() {
	_, err := testApp.Delete()
	Expect(err).NotTo(HaveOccurred())
})
