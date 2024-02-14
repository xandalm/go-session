package session

import (
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSave(t *testing.T) {
	storage := &MemoryStorage{
		sessions: map[string]ISession{},
	}
	t.Run("save session", func(t *testing.T) {
		sess := &stubSession{
			Id: "1",
		}

		err := storage.Save(sess)

		assert.NoError(t, err)

		if storage.sessions[sess.Id] != sess {
			t.Errorf("didn't save session")
		}
	})
}
