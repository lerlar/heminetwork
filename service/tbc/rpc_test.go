// Copyright (c) 2024 Hemi Labs, Inc.
// Use of this source code is governed by the MIT License,
// which can be found in the LICENSE file.

package tbc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/davecgh/go-spew/spew"
	"github.com/docker/go-connections/nat"
	"github.com/go-test/deep"
	"github.com/testcontainers/testcontainers-go"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/hemilabs/heminetwork/api/protocol"
	"github.com/hemilabs/heminetwork/api/tbcapi"
	"github.com/hemilabs/heminetwork/bitcoin"
	"github.com/hemilabs/heminetwork/database/tbcd"
)

func TestBlockHeadersByHeightRaw(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 100, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()
	_, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.BlockHeadersByHeightRawResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		lastErr = nil
		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.BlockHeadersByHeightRawRequest{
			Height: 55,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdBlockHeadersByHeightRawResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}
			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}
	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}

	bh, err := bytes2Header(response.BlockHeaders[0])
	if err != nil {
		t.Fatal(err)
	}

	t.Logf(spew.Sdump(bh))

	if response.Error != nil {
		t.Errorf("got unwanted error: %v", response.Error)
	}

	cliBlockHeader := bitcoindBlockAtHeight(ctx, t, bitcoindContainer, 55)
	expected := cliBlockHeaderToRaw(t, cliBlockHeader)
	if diff := deep.Equal(expected, response.BlockHeaders); len(diff) > 0 {
		t.Errorf("unexpected diff: %s", diff)
	}
}

func TestBlockHeadersByHeight(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 100, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	_, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.BlockHeadersByHeightResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		lastErr = nil
		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.BlockHeadersByHeightRequest{
			Height: 55,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdBlockHeadersByHeightResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}
			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}

	if response.Error != nil {
		t.Errorf("got unwanted error: %v", response.Error)
	}

	cliBlockHeader := bitcoindBlockAtHeight(ctx, t, bitcoindContainer, 55)
	expected := cliBlockHeaderToTBC(t, cliBlockHeader)
	if diff := deep.Equal(expected, response.BlockHeaders); len(diff) > 0 {
		t.Errorf("unexpected diff: %s", diff)
	}
}

func TestBlockHeadersByHeightDoesNotExist(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 100, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	_, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.BlockHeadersByHeightResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		lastErr = nil
		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.BlockHeadersByHeightRequest{
			Height: 550,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdBlockHeadersByHeightResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}
			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}

	if response.Error.Message != "block headers not found at height 550" {
		t.Fatalf("unexpected error message: %s", response.Error.Message)
	}
}

func TestBlockHeadersBestRaw(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 50, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	_, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.BlockHeadersBestRawResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		lastErr = nil
		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.BlockHeadersBestRawRequest{})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdBlockHeadersBestRawResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}
			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}
	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}

	bh, err := bytes2Header(response.BlockHeaders[0])
	if err != nil {
		t.Fatal(err)
	}

	t.Logf(spew.Sdump(bh))

	if response.Error != nil {
		t.Errorf("got unwanted error: %v", response.Error)
	}

	cliBlockHeader := bitcoindBestBlock(ctx, t, bitcoindContainer)
	expected := cliBlockHeaderToRaw(t, cliBlockHeader)
	if diff := deep.Equal(expected, response.BlockHeaders); len(diff) > 0 {
		t.Errorf("unexpected diff: %s", diff)
	}
}

func TestBtcBlockHeadersBest(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 100, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	_, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.BlockHeadersBestResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		lastErr = nil
		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.BlockHeadersBestRequest{})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdBlockHeadersBestResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}
			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}

	if response.Error != nil {
		t.Errorf("got unwanted error: %v", response.Error)
	}

	cliBlockHeader := bitcoindBestBlock(ctx, t, bitcoindContainer)
	expected := cliBlockHeaderToTBC(t, cliBlockHeader)
	if diff := deep.Equal(expected, response.BlockHeaders); len(diff) > 0 {
		t.Errorf("unexpected diff: %s", diff)
	}
}

func TestBalanceByAddress(t *testing.T) {
	skipIfNoDocker(t)

	type testTableItem struct {
		name          string
		address       func() string
		doNotGenerate bool
	}

	testTable := []testTableItem{
		{
			name: "Pay to public key hash",
			address: func() string {
				_, _, address, err := bitcoin.KeysAndAddressFromHexString(
					privateKey,
					&chaincfg.RegressionNetParams,
				)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
		},
		{
			name: "Pay to script hash",
			address: func() string {
				address, err := btcutil.NewAddressScriptHash([]byte("blahblahscripthash"), &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
		},
		{
			name: "Pay to witness public key hash",
			address: func() string {
				address, err := btcutil.NewAddressWitnessPubKeyHash([]byte("blahblahwitnesspublickeyhash")[:20], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
		},
		{
			name: "Pay to witness script hash",
			address: func() string {
				address, err := btcutil.NewAddressWitnessScriptHash([]byte("blahblahwitnessscripthashblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
		},
		{
			name: "Pay to taproot",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
		},
		{
			name: "no balance",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			doNotGenerate: true,
		},
	}

	for _, tti := range testTable {
		t.Run(tti.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			initialBlocks := 0
			if !tti.doNotGenerate {
				initialBlocks = 4
			}

			bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, uint64(initialBlocks), tti.address())
			defer func() {
				if err := bitcoindContainer.Terminate(ctx); err != nil {
					panic(err)
				}
			}()

			// generate to another address to ensure it's not included in our query
			someOtherAddress, err := btcutil.NewAddressScriptHash([]byte("blahblahotherscripthash"), &chaincfg.RegressionNetParams)
			if err != nil {
				t.Fatal(err)
			}
			_, err = runBitcoinCommand(
				ctx,
				t,
				bitcoindContainer,
				[]string{
					"bitcoin-cli",
					"-regtest=1",
					"generatetoaddress",
					"3",
					someOtherAddress.EncodeAddress(),
				})
			if err != nil {
				t.Fatal(err)
			}

			tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

			c, _, err := websocket.Dial(ctx, tbcUrl, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()

			assertPing(ctx, t, c, tbcapi.CmdPingRequest)

			tws := &tbcWs{
				conn: protocol.NewWSConn(c),
			}

			var lastErr error
			var response tbcapi.BalanceByAddressResponse
			for {
				select {
				case <-time.After(1 * time.Second):
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				err = tbcServer.UtxoIndexer(ctx, 0, 1000)
				if err != nil {
					t.Fatal(err)
				}
				lastErr = nil
				err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.BalanceByAddressRequest{
					Address: tti.address(),
				})
				if err != nil {
					lastErr = err
					continue
				}

				var v protocol.Message
				err = wsjson.Read(ctx, c, &v)
				if err != nil {
					lastErr = err
					continue
				}

				if v.Header.Command == tbcapi.CmdBalanceByAddressResponse {
					if err := json.Unmarshal(v.Payload, &response); err != nil {
						t.Fatal(err)
					}

					var pricePerBlock uint64 = 50 * 100000000
					var blocks uint64 = 4
					var expectedBalance uint64 = 0
					if !tti.doNotGenerate {
						expectedBalance = pricePerBlock * blocks
					}

					expected := tbcapi.BalanceByAddressResponse{
						Balance: expectedBalance,
						Error:   nil,
					}
					if diff := deep.Equal(expected, response); len(diff) > 0 {
						if response.Error != nil {
							t.Error(response.Error.Message)
						}
						t.Logf("unexpected diff: %s", diff)

						// there is a chance we just haven't finished indexing
						// the blocks and txs, retry until timeout
						continue
					}
					break
				} else {
					lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
				}

			}

			if lastErr != nil {
				t.Fatal(lastErr)
			}
		})
	}
}

func TestUtxosByAddressRaw(t *testing.T) {
	skipIfNoDocker(t)

	type testTableItem struct {
		name          string
		address       func() string
		doNotGenerate bool
		limit         uint64
		start         uint64
	}

	testTable := []testTableItem{
		{
			name: "Pay to public key hash",
			address: func() string {
				_, _, address, err := bitcoin.KeysAndAddressFromHexString(
					privateKey,
					&chaincfg.RegressionNetParams,
				)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to script hash",
			address: func() string {
				address, err := btcutil.NewAddressScriptHash([]byte("blahblahscripthash"), &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to witness public key hash",
			address: func() string {
				address, err := btcutil.NewAddressWitnessPubKeyHash([]byte("blahblahwitnesspublickeyhash")[:20], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to witness script hash",
			address: func() string {
				address, err := btcutil.NewAddressWitnessScriptHash([]byte("blahblahwitnessscripthashblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to taproot",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "no balance",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			doNotGenerate: true,
			limit:         10,
		},
		{
			name: "small limit",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblahsmalllimit")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 2,
		},
		{
			name: "offset",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblahsmalllimit")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			start: 3,
			limit: 10,
		},
	}

	for _, tti := range testTable {
		t.Run(tti.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			var bitcoindContainer testcontainers.Container
			var mappedPeerPort nat.Port
			initialBlocks := 0
			if !tti.doNotGenerate {
				initialBlocks = 4
			}
			bitcoindContainer, mappedPeerPort = createBitcoindWithInitialBlocks(ctx, t, uint64(initialBlocks), tti.address())
			defer func() {
				if err := bitcoindContainer.Terminate(ctx); err != nil {
					panic(err)
				}
			}()

			// generate to another address to ensure it's not included in our query
			someOtherAddress, err := btcutil.NewAddressScriptHash([]byte("blahblahotherscripthash"), &chaincfg.RegressionNetParams)
			if err != nil {
				t.Fatal(err)
			}
			_, err = runBitcoinCommand(
				ctx,
				t,
				bitcoindContainer,
				[]string{
					"bitcoin-cli",
					"-regtest=1",
					"generatetoaddress",
					"3",
					someOtherAddress.EncodeAddress(),
				})
			if err != nil {
				t.Fatal(err)
			}

			tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

			c, _, err := websocket.Dial(ctx, tbcUrl, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()

			assertPing(ctx, t, c, tbcapi.CmdPingRequest)

			tws := &tbcWs{
				conn: protocol.NewWSConn(c),
			}

			var lastErr error
			var response tbcapi.UtxosByAddressRawResponse
			for {
				select {
				case <-time.After(1 * time.Second):
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				err = tbcServer.UtxoIndexer(ctx, 0, 1000)
				if err != nil {
					t.Fatal(err)
				}
				lastErr = nil
				err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.UtxosByAddressRawRequest{
					Address: tti.address(),
					Start:   uint(tti.start),
					Count:   uint(tti.limit),
				})
				if err != nil {
					lastErr = err
					continue
				}

				var v protocol.Message
				err = wsjson.Read(ctx, c, &v)
				if err != nil {
					lastErr = err
					continue
				}

				if v.Header.Command == tbcapi.CmdUtxosByAddressRawResponse {
					if err := json.Unmarshal(v.Payload, &response); err != nil {
						t.Fatal(err)
					}

					// we generated 4 blocks to this address previously, therefore
					// there should be 4 utxos
					expectedCount := 4 - tti.start
					if tti.limit < uint64(expectedCount) {
						expectedCount = tti.limit
					}

					if !tti.doNotGenerate && len(response.Utxos) != int(expectedCount) {
						t.Fatalf("should have %d utxos, received: %d", expectedCount, len(response.Utxos))
					} else if tti.doNotGenerate && len(response.Utxos) != 0 {
						t.Fatalf("did not generate any blocks for address, should not have utxos")
					}
					break
				} else {
					lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
				}

			}

			if lastErr != nil {
				t.Fatal(lastErr)
			}
		})
	}
}

func TestUtxosByAddress(t *testing.T) {
	skipIfNoDocker(t)

	type testTableItem struct {
		name          string
		address       func() string
		doNotGenerate bool
		limit         uint64
		start         uint64
	}

	testTable := []testTableItem{
		{
			name: "Pay to public key hash",
			address: func() string {
				_, _, address, err := bitcoin.KeysAndAddressFromHexString(
					privateKey,
					&chaincfg.RegressionNetParams,
				)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to script hash",
			address: func() string {
				address, err := btcutil.NewAddressScriptHash([]byte("blahblahscripthash"), &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to witness public key hash",
			address: func() string {
				address, err := btcutil.NewAddressWitnessPubKeyHash([]byte("blahblahwitnesspublickeyhash")[:20], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to witness script hash",
			address: func() string {
				address, err := btcutil.NewAddressWitnessScriptHash([]byte("blahblahwitnessscripthashblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "Pay to taproot",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 10,
		},
		{
			name: "no balance",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblah")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			doNotGenerate: true,
			limit:         10,
		},
		{
			name: "small limit",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblahsmalllimit")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			limit: 2,
		},
		{
			name: "offset",
			address: func() string {
				address, err := btcutil.NewAddressTaproot([]byte("blahblahwtaprootblahblahblahblahsmalllimit")[:32], &chaincfg.RegressionNetParams)
				if err != nil {
					t.Fatal(err)
				}

				return address.EncodeAddress()
			},
			start: 3,
			limit: 10,
		},
	}

	for _, tti := range testTable {
		t.Run(tti.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			var bitcoindContainer testcontainers.Container
			var mappedPeerPort nat.Port
			initialBlocks := 0
			if !tti.doNotGenerate {
				initialBlocks = 4
			}
			bitcoindContainer, mappedPeerPort = createBitcoindWithInitialBlocks(ctx, t, uint64(initialBlocks), tti.address())
			defer func() {
				if err := bitcoindContainer.Terminate(ctx); err != nil {
					panic(err)
				}
			}()

			// generate to another address to ensure it's not included in our query
			someOtherAddress, err := btcutil.NewAddressScriptHash([]byte("blahblahotherscripthash"), &chaincfg.RegressionNetParams)
			if err != nil {
				t.Fatal(err)
			}
			_, err = runBitcoinCommand(
				ctx,
				t,
				bitcoindContainer,
				[]string{
					"bitcoin-cli",
					"-regtest=1",
					"generatetoaddress",
					"3",
					someOtherAddress.EncodeAddress(),
				})
			if err != nil {
				t.Fatal(err)
			}

			tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

			c, _, err := websocket.Dial(ctx, tbcUrl, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()

			assertPing(ctx, t, c, tbcapi.CmdPingRequest)

			tws := &tbcWs{
				conn: protocol.NewWSConn(c),
			}

			var lastErr error
			var response tbcapi.UtxosByAddressResponse
			for {
				select {
				case <-time.After(1 * time.Second):
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				err = tbcServer.UtxoIndexer(ctx, 0, 1000)
				if err != nil {
					t.Fatal(err)
				}
				lastErr = nil
				err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.UtxosByAddressRequest{
					Address: tti.address(),
					Start:   uint(tti.start),
					Count:   uint(tti.limit),
				})
				if err != nil {
					lastErr = err
					continue
				}

				var v protocol.Message
				err = wsjson.Read(ctx, c, &v)
				if err != nil {
					lastErr = err
					continue
				}

				if v.Header.Command == tbcapi.CmdUtxosByAddressResponse {
					if err := json.Unmarshal(v.Payload, &response); err != nil {
						t.Fatal(err)
					}

					// we generated 4 blocks to this address previously, therefore
					// there should be 4 utxos
					expectedCount := 4 - tti.start
					if tti.limit < uint64(expectedCount) {
						expectedCount = tti.limit
					}

					if !tti.doNotGenerate && len(response.Utxos) != int(expectedCount) {
						t.Fatalf("should have %d utxos, received: %d", expectedCount, len(response.Utxos))
					} else if tti.doNotGenerate && len(response.Utxos) != 0 {
						t.Fatalf("did not generate any blocks for address, should not have utxos")
					}
					break
				} else {
					lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
				}

			}

			if lastErr != nil {
				t.Fatal(lastErr)
			}
		})
	}
}

func TestTxByIdRaw(t *testing.T) {
	skipIfNoDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, _, address, err := bitcoin.KeysAndAddressFromHexString(
		privateKey,
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		t.Fatal(err)
	}

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 4, address.String())
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.TxByIdRawResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		err = tbcServer.TxIndexer(ctx, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		lastErr = nil
		txId := getRandomTxId(ctx, t, bitcoindContainer)
		txIdBytes, err := hex.DecodeString(txId)
		if err != nil {
			t.Fatal(err)
		}

		slices.Reverse(txIdBytes) // convert to natural order

		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.TxByIdRawRequest{
			TxId: txIdBytes,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdTxByIdRawResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}

			if response.Error != nil {
				t.Fatal(response.Error.Message)
			}

			// XXX - write a better test than this, we should be able to compare
			// against bitcoin-cli response fields

			// did we get the tx and can we parse it?
			tx, err := bytes2Tx(response.Tx)
			if err != nil {
				t.Fatal(err)
			}

			// is the hash equal to what we queried for?
			if tx.TxHash().String() != txId {
				t.Fatalf("id mismatch: %s != %s", tx.TxHash().String(), txId)
			}

			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
}

func TestTxByIdRawInvalid(t *testing.T) {
	skipIfNoDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	_, _, address, err := bitcoin.KeysAndAddressFromHexString(
		privateKey,
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		t.Fatal(err)
	}

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 4, address.String())
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.TxByIdRawResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		err = tbcServer.TxIndexer(ctx, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		lastErr = nil
		txId := getRandomTxId(ctx, t, bitcoindContainer)
		txIdBytes, err := hex.DecodeString(txId)
		if err != nil {
			t.Fatal(err)
		}

		txIdBytes[0]++

		slices.Reverse(txIdBytes) // convert to natural order

		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.TxByIdRawRequest{
			TxId: txIdBytes,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdTxByIdRawResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}

			if response.Error == nil {
				t.Fatal("expecting error")
			}

			if response.Error != nil {
				if !strings.Contains(response.Error.Message, "not found:") {
					t.Fatalf("incorrect error found %s", response.Error.Message)
				}
			}

			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
}

func TestTxByIdRawNotFound(t *testing.T) {
	skipIfNoDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 0, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	_, _, address, err := bitcoin.KeysAndAddressFromHexString(
		privateKey,
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = runBitcoinCommand(
		ctx,
		t,
		bitcoindContainer,
		[]string{
			"bitcoin-cli",
			"-regtest=1",
			"generatetoaddress",
			"4",
			address.EncodeAddress(),
		})
	if err != nil {
		t.Fatal(err)
	}

	tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.TxByIdRawResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		err = tbcServer.TxIndexer(ctx, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		lastErr = nil
		txId := getRandomTxId(ctx, t, bitcoindContainer)
		txIdBytes, err := hex.DecodeString(txId)
		if err != nil {
			t.Fatal(err)
		}

		txIdBytes = append(txIdBytes, 8)

		slices.Reverse(txIdBytes) // convert to natural order

		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.TxByIdRawRequest{
			TxId: txIdBytes,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdTxByIdRawResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}

			if response.Error == nil {
				t.Fatal("expecting error")
			}

			if response.Error != nil {
				if !strings.Contains(response.Error.Message, "invalid tx id") {
					t.Fatalf("incorrect error found: %s", response.Error.Message)
				}
			}

			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
}

func TestTxById(t *testing.T) {
	skipIfNoDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, _, address, err := bitcoin.KeysAndAddressFromHexString(
		privateKey,
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		t.Fatal(err)
	}

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 4, address.String())
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.TxByIdResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		err = tbcServer.TxIndexer(ctx, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		lastErr = nil
		txId := getRandomTxId(ctx, t, bitcoindContainer)
		txIdBytes, err := hex.DecodeString(txId)
		if err != nil {
			t.Fatal(err)
		}

		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.TxByIdRequest{
			TxId: txIdBytes,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdTxByIdResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}

			if response.Error != nil {
				t.Fatal(response.Error.Message)
			}

			tx, err := tbcServer.TxById(ctx, tbcd.TxId(reverseBytes(txIdBytes)))
			if err != nil {
				t.Fatal(err)
			}

			w := wireTxToTBC(tx)

			if diff := deep.Equal(w, response.Tx); len(diff) > 0 {
				t.Fatal(diff)
			}

			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
}

func TestTxByIdInvalid(t *testing.T) {
	skipIfNoDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	_, _, address, err := bitcoin.KeysAndAddressFromHexString(
		privateKey,
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		t.Fatal(err)
	}

	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 4, address.String())
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.TxByIdResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		err = tbcServer.TxIndexer(ctx, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		lastErr = nil
		txId := getRandomTxId(ctx, t, bitcoindContainer)
		txIdBytes, err := hex.DecodeString(txId)
		if err != nil {
			t.Fatal(err)
		}

		txIdBytes[0]++

		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.TxByIdRequest{
			TxId: txIdBytes,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdTxByIdResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}

			if response.Error == nil {
				t.Fatal("expecting error")
			}

			if response.Error != nil {
				if !strings.Contains(response.Error.Message, "not found:") {
					t.Fatalf("incorrect error found %s", response.Error.Message)
				}
			}

			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
}

func TestTxByIdNotFound(t *testing.T) {
	skipIfNoDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	bitcoindContainer, mappedPeerPort := createBitcoindWithInitialBlocks(ctx, t, 0, "")
	defer func() {
		if err := bitcoindContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	_, _, address, err := bitcoin.KeysAndAddressFromHexString(
		privateKey,
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = runBitcoinCommand(
		ctx,
		t,
		bitcoindContainer,
		[]string{
			"bitcoin-cli",
			"-regtest=1",
			"generatetoaddress",
			"4",
			address.EncodeAddress(),
		})
	if err != nil {
		t.Fatal(err)
	}

	tbcServer, tbcUrl := createTbcServer(ctx, t, mappedPeerPort)

	c, _, err := websocket.Dial(ctx, tbcUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	assertPing(ctx, t, c, tbcapi.CmdPingRequest)

	tws := &tbcWs{
		conn: protocol.NewWSConn(c),
	}

	var lastErr error
	var response tbcapi.TxByIdResponse
	for {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		err = tbcServer.TxIndexer(ctx, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		lastErr = nil
		txId := getRandomTxId(ctx, t, bitcoindContainer)
		txIdBytes, err := hex.DecodeString(txId)
		if err != nil {
			t.Fatal(err)
		}

		txIdBytes = append(txIdBytes, 8)

		err = tbcapi.Write(ctx, tws.conn, "someid", tbcapi.TxByIdRequest{
			TxId: txIdBytes,
		})
		if err != nil {
			lastErr = err
			continue
		}

		var v protocol.Message
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			lastErr = err
			continue
		}

		if v.Header.Command == tbcapi.CmdTxByIdResponse {
			if err := json.Unmarshal(v.Payload, &response); err != nil {
				t.Fatal(err)
			}

			if response.Error == nil {
				t.Fatal("expecting error")
			}

			if response.Error != nil {
				if !strings.Contains(response.Error.Message, "invalid tx id") {
					t.Fatalf("incorrect error found: %s", response.Error.Message)
				}
			}

			break
		} else {
			lastErr = fmt.Errorf("received unexpected command: %s", v.Header.Command)
		}

	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
}

func assertPing(ctx context.Context, t *testing.T, c *websocket.Conn, cmd protocol.Command) {
	var v protocol.Message
	err := wsjson.Read(ctx, c, &v)
	if err != nil {
		t.Fatal(err)
	}

	if v.Header.Command != cmd {
		t.Fatalf("unexpected command: %s", v.Header.Command)
	}
}
