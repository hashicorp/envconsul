package dependency

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
)

func TestNewVaultTokenQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		exp  *VaultTokenQuery
		err  bool
	}{
		{
			"default",
			&VaultTokenQuery{},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewVaultTokenQuery()
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestVaultTokenQuery_Fetch(t *testing.T) {
	t.Parallel()

	clients, server := testVaultServer(t)
	defer server.Stop()

	// Grab the underlying client
	vault := clients.vault.client

	// Create a new token - the default token is a root token and is therefore
	// not renewable
	secret, err := vault.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}
	vault.SetToken(secret.Auth.ClientToken)

	t.Run("fetches", func(t *testing.T) {
		d, err := NewVaultTokenQuery()
		if err != nil {
			t.Fatal(err)
		}

		data, _, err := d.Fetch(clients, nil)
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := data.(*Secret); !ok {
			t.Error("not *Secret")
		}
	})

	t.Run("stops", func(t *testing.T) {
		d, err := NewVaultTokenQuery()
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(clients, nil)
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}

		d.Stop()

		select {
		case err := <-errCh:
			if err != ErrStopped {
				t.Fatal(err)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("did not stop")
		}
	})

	t.Run("fires_changes", func(t *testing.T) {
		d, err := NewVaultTokenQuery()
		if err != nil {
			t.Fatal(err)
		}

		_, qm, err := d.Fetch(clients, nil)
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(clients, &QueryOptions{WaitIndex: qm.LastIndex})
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
				return
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}
	})
}

func TestVaultTokenQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		exp  string
	}{
		{
			"default",
			"vault.token",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultTokenQuery()
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
