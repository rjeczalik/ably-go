// +build ignore

package ably_test

import (
	"github.com/ably/ably-go/ably"

	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/gomega"
)

var _ = Describe("Auth", func() {
	Describe("RequestToken", func() {
		It("gets a token from the API", func() {
			params := &ably.TokenParams{
				TTL:        60 * 60 * 1000,
				Capability: ably.Capability{"foo": []string{"publish"}},
				ClientID:   "client_string",
			}
			keyName, _ := testApp.KeyParts()
			token, err := client.Auth.RequestToken(nil, params)

			Expect(err).NotTo(HaveOccurred())
			Expect(token.Token).To(ContainSubstring(testApp.Config.AppID))
			Expect(token.KeyName).To(Equal(keyName))
			Expect(token.Issued).NotTo(Equal(int64(0)))
			Expect(token.Capability).To(Equal(params.Capability))
		})
	})

	Describe("CreateTokenRequest", func() {
		It("gets a token from the API", func() {
			params := &ably.TokenParams{
				TTL:        60 * 60 * 1000,
				Capability: ably.Capability{"foo": []string{"publish"}},
			}
			req, err := client.Auth.CreateTokenRequest(nil, params)

			Expect(err).NotTo(HaveOccurred())
			Expect(req.KeyName).To(ContainSubstring(testApp.Config.AppID))
			Expect(req.Mac).NotTo(BeNil())
		})
	})
})
