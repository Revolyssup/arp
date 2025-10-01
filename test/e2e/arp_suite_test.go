package arp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Arp Suite")
}
