package cmd_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"scuffinger/cmd"
)

var _ = Describe("Version Command", func() {
	It("should print the version string", func() {
		// Set a known version for testing
		cmd.Version = "1.2.3"

		// Create root command and capture output
		root := cmd.NewRootCommand()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetArgs([]string{"version"})

		err := root.Execute()
		Expect(err).ToNot(HaveOccurred())
		Expect(buf.String()).To(ContainSubstring("1.2.3"))
	})
})
