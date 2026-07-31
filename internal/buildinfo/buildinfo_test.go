package buildinfo_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"j30att/observer/internal/buildinfo"
)

func TestPrint(t *testing.T) {
	t.Run("Должен выполниться без ошибок", func(t *testing.T) {
		var output bytes.Buffer

		buildinfo.Print(&output, "v1.0.0", "2026-07-31", "abc123")

		assert.Equal(t, "Build version: v1.0.0\nBuild date: 2026-07-31\nBuild commit: abc123\n", output.String())
	})

	t.Run("Должен вывести N/A для пустых значений", func(t *testing.T) {
		var output bytes.Buffer

		buildinfo.Print(&output, "", "", "")

		assert.Equal(t, "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n", output.String())
	})
}
