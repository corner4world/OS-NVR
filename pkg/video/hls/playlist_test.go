package hls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextSegment(t *testing.T) {
	playlist := newPlaylist(0, 3)

	seg5 := &SegmentFinalized{ID: 5}
	seg6 := &SegmentFinalized{ID: 6}

	playlist.onSegmentFinalized(seg5)
	playlist.onSegmentFinalized(seg6)

	cases := map[string]struct {
		prevID   uint64
		expected *SegmentFinalized
	}{
		"before": {3, seg5},
		"ok":     {4, seg5},
		"ok2":    {5, seg6},
		"after":  {7, seg5},
		"after2": {999, seg5},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			seg, err := playlist.nextSegment(&SegmentFinalized{ID: tc.prevID})
			require.NoError(t, err)
			require.Equal(t, tc.expected, seg)
		})
	}
	t.Run("blocking", func(t *testing.T) {
		seg7 := &SegmentFinalized{ID: 7}
		done := make(chan struct{})
		go func() {
			seg, err := playlist.nextSegment(&SegmentFinalized{ID: 6})
			require.NoError(t, err)
			require.Equal(t, seg7, seg)
			close(done)
		}()

		playlist.onSegmentFinalized(seg7)
		<-done
	})
}
