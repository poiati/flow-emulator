package examples

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/flow-go-sdk"
	"github.com/dapperlabs/flow-go-sdk/keys"
)

const (
	MarketContractFile = "./contracts/market.cdc"
)

func TestMarketDeployment(t *testing.T) {
	b := NewEmulator()

	// Should be able to deploy a contract as a new account with no keys.
	tokenCode := ReadFile(resourceTokenContractFile)
	_, err := b.CreateAccount(nil, tokenCode, GetNonce())
	if !assert.Nil(t, err) {
		t.Log(err.Error())
	}
	_, err = b.CommitBlock()
	require.NoError(t, err)

	nftCode := ReadFile(NFTContractFile)
	_, err = b.CreateAccount(nil, nftCode, GetNonce())
	if !assert.Nil(t, err) {
		t.Log(err.Error())
	}
	_, err = b.CommitBlock()
	require.NoError(t, err)

	// Should be able to deploy a contract as a new account with no keys.
	marketCode := ReadFile(MarketContractFile)
	_, err = b.CreateAccount(nil, marketCode, GetNonce())
	if !assert.Nil(t, err) {
		t.Log(err.Error())
	}
	_, err = b.CommitBlock()
	require.NoError(t, err)
}

func TestCreateSale(t *testing.T) {
	b := NewEmulator()

	// first deploy the FT, NFT, and market code
	tokenCode := ReadFile(resourceTokenContractFile)
	tokenAddr, err := b.CreateAccount(nil, tokenCode, GetNonce())
	assert.Nil(t, err)
	_, err = b.CommitBlock()
	require.NoError(t, err)

	nftCode := ReadFile(NFTContractFile)
	nftAddr, err := b.CreateAccount(nil, nftCode, GetNonce())
	assert.Nil(t, err)
	_, err = b.CommitBlock()
	require.NoError(t, err)

	marketCode := ReadFile(MarketContractFile)
	marketAddr, err := b.CreateAccount(nil, marketCode, GetNonce())
	assert.Nil(t, err)
	_, err = b.CommitBlock()
	require.NoError(t, err)

	// create two new accounts
	bastianPrivateKey := RandomPrivateKey()
	bastianPublicKey := bastianPrivateKey.ToAccountKey()
	bastianPublicKey.Weight = keys.PublicKeyWeightThreshold

	bastianAddress, err := b.CreateAccount([]flow.AccountKey{bastianPublicKey}, nil, GetNonce())

	joshPrivateKey := RandomPrivateKey()
	joshPublicKey := joshPrivateKey.ToAccountKey()
	joshPublicKey.Weight = keys.PublicKeyWeightThreshold

	joshAddress, err := b.CreateAccount([]flow.AccountKey{joshPublicKey}, nil, GetNonce())

	t.Run("Should be able to create FTs and NFT collections in each accounts storage", func(t *testing.T) {
		// create Fungible tokens and NFTs in each accounts storage and store references
		setupUsersTokens(t, b, tokenAddr, nftAddr, []flow.AccountPrivateKey{bastianPrivateKey, joshPrivateKey}, []flow.Address{bastianAddress, joshAddress})
	})

	t.Run("Can create sale collection", func(t *testing.T) {
		tx := flow.NewTransaction().
			SetScript(GenerateCreateSaleScript(tokenAddr, marketAddr)).
			SetGasLimit(10).
			SetProposalKey(b.RootKey().Address, b.RootKey().ID, b.RootKey().SequenceNumber).
			SetPayer(b.RootKey().Address, b.RootKey().ID).
			AddAuthorizer(bastianAddress, 0)

		SignAndSubmit(t, b, tx, []flow.AccountPrivateKey{b.RootKey().PrivateKey, bastianPrivateKey}, []flow.Address{b.RootAccountAddress(), bastianAddress}, false)
	})

	t.Run("Can put an NFT up for sale", func(t *testing.T) {
		tx := flow.NewTransaction().
			SetScript(GenerateStartSaleScript(nftAddr, marketAddr, 1, 10)).
			SetGasLimit(10).
			SetProposalKey(b.RootKey().Address, b.RootKey().ID, b.RootKey().SequenceNumber).
			SetPayer(b.RootKey().Address, b.RootKey().ID).
			AddAuthorizer(bastianAddress, 0)

		SignAndSubmit(t, b, tx, []flow.AccountPrivateKey{b.RootKey().PrivateKey, bastianPrivateKey}, []flow.Address{b.RootAccountAddress(), bastianAddress}, false)
	})

	t.Run("Cannot buy an NFT for less than the sale price", func(t *testing.T) {
		tx := flow.NewTransaction().
			SetScript(GenerateBuySaleScript(tokenAddr, nftAddr, marketAddr, bastianAddress, 1, 9)).
			SetGasLimit(10).
			SetProposalKey(b.RootKey().Address, b.RootKey().ID, b.RootKey().SequenceNumber).
			SetPayer(b.RootKey().Address, b.RootKey().ID).
			AddAuthorizer(joshAddress, 0)

		SignAndSubmit(t, b, tx, []flow.AccountPrivateKey{b.RootKey().PrivateKey, joshPrivateKey}, []flow.Address{b.RootAccountAddress(), joshAddress}, true)
	})

	t.Run("Cannot buy an NFT that is not for sale", func(t *testing.T) {
		tx := flow.NewTransaction().
			SetScript(GenerateBuySaleScript(tokenAddr, nftAddr, marketAddr, bastianAddress, 2, 10)).
			SetGasLimit(10).
			SetProposalKey(b.RootKey().Address, b.RootKey().ID, b.RootKey().SequenceNumber).
			SetPayer(b.RootKey().Address, b.RootKey().ID).
			AddAuthorizer(joshAddress, 0)

		SignAndSubmit(t, b, tx, []flow.AccountPrivateKey{b.RootKey().PrivateKey, joshPrivateKey}, []flow.Address{b.RootAccountAddress(), joshAddress}, true)
	})

	t.Run("Can buy an NFT that is for sale", func(t *testing.T) {
		tx := flow.NewTransaction().
			SetScript(GenerateBuySaleScript(tokenAddr, nftAddr, marketAddr, bastianAddress, 1, 10)).
			SetGasLimit(10).
			SetProposalKey(b.RootKey().Address, b.RootKey().ID, b.RootKey().SequenceNumber).
			SetPayer(b.RootKey().Address, b.RootKey().ID).
			AddAuthorizer(joshAddress, 0)

		SignAndSubmit(t, b, tx, []flow.AccountPrivateKey{b.RootKey().PrivateKey, joshPrivateKey}, []flow.Address{b.RootAccountAddress(), joshAddress}, false)

		result, err := b.ExecuteScript(GenerateInspectVaultScript(tokenAddr, bastianAddress, 40))
		require.NoError(t, err)
		if !assert.True(t, result.Succeeded()) {
			t.Log(result.Error.Error())
		}

		result, err = b.ExecuteScript(GenerateInspectVaultScript(tokenAddr, joshAddress, 20))
		require.NoError(t, err)
		if !assert.True(t, result.Succeeded()) {
			t.Log(result.Error.Error())
		}

		// Assert that the accounts' collections are correct
		result, err = b.ExecuteScript(GenerateInspectCollectionScript(nftAddr, bastianAddress, 1, false))
		require.NoError(t, err)
		if !assert.True(t, result.Succeeded()) {
			t.Log(result.Error.Error())
		}

		result, err = b.ExecuteScript(GenerateInspectCollectionScript(nftAddr, bastianAddress, 2, false))
		require.NoError(t, err)
		if !assert.True(t, result.Succeeded()) {
			t.Log(result.Error.Error())
		}

		result, err = b.ExecuteScript(GenerateInspectCollectionScript(nftAddr, joshAddress, 1, true))
		require.NoError(t, err)
		if !assert.True(t, result.Succeeded()) {
			t.Log(result.Error.Error())
		}

		result, err = b.ExecuteScript(GenerateInspectCollectionScript(nftAddr, joshAddress, 2, true))
		require.NoError(t, err)
		if !assert.True(t, result.Succeeded()) {
			t.Log(result.Error.Error())
		}
	})

}
