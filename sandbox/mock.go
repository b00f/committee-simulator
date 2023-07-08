package sandbox

import (
	"github.com/pactus-project/pactus/committee"
	"github.com/pactus-project/pactus/crypto"
	"github.com/pactus-project/pactus/crypto/bls"
	"github.com/pactus-project/pactus/crypto/hash"
	"github.com/pactus-project/pactus/sandbox"
	"github.com/pactus-project/pactus/sortition"
	"github.com/pactus-project/pactus/types/account"
	"github.com/pactus-project/pactus/types/param"
	"github.com/pactus-project/pactus/types/validator"
)

var _ sandbox.Sandbox = &MockSandbox{}

// MockSandbox is a testing mock for sandbox.
type MockSandbox struct {
	MockCommittee    committee.Committee
	MockValidators   map[crypto.Address]*validator.Validator
	MockParams       param.Params
	BlockHashes      []hash.Hash
	StartHeight      uint32
	JoinedValidators map[crypto.Address]bool
}

func (m *MockSandbox) Account(addr crypto.Address) *account.Account {
	return nil
}
func (m *MockSandbox) MakeNewAccount(_ crypto.Address) *account.Account {
	return nil
}
func (m *MockSandbox) UpdateAccount(addr crypto.Address, acc *account.Account) {

}
func (m *MockSandbox) Validator(addr crypto.Address) *validator.Validator {
	val := m.MockValidators[addr]
	return val
}
func (m *MockSandbox) JoinedToCommittee(addr crypto.Address) {
	m.JoinedValidators[addr] = true
}
func (m *MockSandbox) IsJoinedCommittee(addr crypto.Address) bool {
	return m.JoinedValidators[addr]
}
func (m *MockSandbox) MakeNewValidator(pub *bls.PublicKey) *validator.Validator {
	return nil
}
func (m *MockSandbox) UpdateValidator(val *validator.Validator) {
	m.MockValidators[val.Address()] = val
}
func (m *MockSandbox) CurrentHeight() uint32 {
	return uint32(len(m.BlockHashes)) + m.StartHeight
}
func (m *MockSandbox) Params() param.Params {
	return m.MockParams
}
func (m *MockSandbox) FindBlockHashByStamp(stamp hash.Stamp) (hash.Hash, bool) {
	for _, h := range m.BlockHashes {
		if h.Stamp().EqualsTo(stamp) {
			return h, true
		}
	}
	return hash.Hash{}, false
}
func (m *MockSandbox) FindBlockHeightByStamp(stamp hash.Stamp) (uint32, bool) {
	for i, h := range m.BlockHashes {
		if h.Stamp().EqualsTo(stamp) {
			return uint32(i) + m.StartHeight, true
		}
	}
	return 0, false
}
func (m *MockSandbox) IterateAccounts(consumer func(crypto.Address, *account.Account, bool)) {

}
func (m *MockSandbox) IterateValidators(consumer func(*validator.Validator, bool, bool)) {
	for _, val := range m.MockValidators {
		consumer(val, false, m.JoinedValidators[val.Address()])
	}
}

func (m *MockSandbox) Committee() committee.Reader {
	return m.MockCommittee
}

func (m *MockSandbox) VerifyProof(hash.Stamp, sortition.Proof, *validator.Validator) bool {
	return true
}

func (m *MockSandbox) Reset() {
	m.JoinedValidators = make(map[crypto.Address]bool)
}
