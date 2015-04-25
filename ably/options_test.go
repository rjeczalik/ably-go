package ably_test

import (
	"net/url"

	"github.com/ably/ably-go/ably"

	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/ably/ably-go/Godeps/_workspace/src/github.com/onsi/gomega"
)

var _ = Describe("ScopeParams", func() {
	var (
		params ably.ScopeParams
		values *url.Values
	)

	Describe("Values", func() {
		BeforeEach(func() {
			params = ably.ScopeParams{}
		})

		Context("with an invalid range", func() {
			BeforeEach(func() {
				params.Start = 123
				params.End = 122
			})

			It("returns an error", func() {
				err := params.EncodeValues(values)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("PaginateParams", func() {
	var (
		params ably.PaginateParams
		values *url.Values
	)

	BeforeEach(func() {
		values = &url.Values{}
		params = ably.PaginateParams{}
	})

	Describe("Values", func() {
		It("returns nil with no values", func() {
			params = ably.PaginateParams{}
			err := params.EncodeValues(values)
			Expect(err).NotTo(HaveOccurred())
			Expect(values.Encode()).To(Equal(""))
		})

		It("returns the full params encoded", func() {
			params = ably.PaginateParams{
				Limit:     1,
				Direction: "backwards",
				ScopeParams: ably.ScopeParams{
					Start: 123,
					End:   124,
					Unit:  "hello",
				},
			}
			err := params.EncodeValues(values)
			Expect(err).NotTo(HaveOccurred())
			Expect(values.Encode()).To(Equal("direction=backwards&end=124&limit=1&start=123&unit=hello"))
		})

		Context("with values", func() {
			BeforeEach(func() {
				params = ably.PaginateParams{
					Limit:     10,
					Direction: "backwards",
				}
			})

			It("does not return an error", func() {
				err := params.EncodeValues(values)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with a value for ScopeParams", func() {
			BeforeEach(func() {
				params.Start = 123
			})

			It("creates a valid url", func() {
				err := params.EncodeValues(values)
				Expect(err).NotTo(HaveOccurred())
				Expect(values.Get("start")).To(Equal("123"))
			})
		})

		Context("with invalid value for direction", func() {
			BeforeEach(func() {
				params.Direction = "unknown"
			})

			It("resets the value to the default", func() {
				err := params.EncodeValues(values)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with invalid value for limit", func() {
			BeforeEach(func() {
				params.Limit = -1
			})

			It("resets the value to the default", func() {
				err := params.EncodeValues(values)
				Expect(err).NotTo(HaveOccurred())
				Expect(values.Get("limit")).To(Equal("100"))
			})
		})
	})
})
